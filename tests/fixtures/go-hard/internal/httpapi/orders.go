package httpapi

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (a *API) GetOrder(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	o, err := a.orders.Get(id)
	if err != nil || o.UserID != c.GetInt64("userID") {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.JSON(http.StatusOK, o)
}

func (a *API) UpdateOrderAddress(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	var body struct {
		Address string `json:"address" binding:"required,max=255"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if err := a.orders.UpdateAddress(id, body.Address); err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.Status(http.StatusNoContent)
}

func (a *API) Invoice(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	o, err := a.orders.Get(id)
	if err != nil || o.UserID != c.GetInt64("userID") {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	tmp, err := os.MkdirTemp("", "invoice")
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmp)
	out := filepath.Join(tmp, "invoice.pdf")
	src := "http://127.0.0.1:8081/render/invoice/" + strconv.FormatInt(o.ID, 10)
	if err := exec.Command("wkhtmltopdf", "--quiet", src, out).Run(); err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.File(out)
}

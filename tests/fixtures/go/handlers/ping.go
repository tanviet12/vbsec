package handlers

import (
	"net/http"
	"os/exec"

	"github.com/gin-gonic/gin"
)

func (h *Handler) Ping(c *gin.Context) {
	host := c.PostForm("host")
	out, err := exec.Command("sh", "-c", "ping -c 1 "+host).CombinedOutput()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ping failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": string(out)})
}

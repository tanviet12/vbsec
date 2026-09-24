package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) SearchUsers(c *gin.Context) {
	name := c.Query("name")
	var users []User
	if err := h.db.Raw("SELECT id, name, email FROM users WHERE name = '" + name + "'").Scan(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, users)
}

package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Session struct {
	Token  string
	UserID uint
}

func (h *Handler) RequireUser(c *gin.Context) {
	token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	var s Session
	if token == "" || h.db.Where("token = ? AND expires_at > NOW()", token).First(&s).Error != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	c.Set("userID", s.UserID)
	c.Next()
}

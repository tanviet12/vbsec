package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) Health(c *gin.Context) {
	sqlDB, err := h.db.DB()
	if err != nil || sqlDB.Ping() != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "down"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

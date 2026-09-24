package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetOrder(c *gin.Context) {
	userID := c.GetUint("userID")
	var order Order
	err := h.db.Where("id = ? AND user_id = ?", c.Param("id"), userID).First(&order).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, order)
}

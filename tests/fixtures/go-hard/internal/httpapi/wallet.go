package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"example.com/wallet/internal/service"
)

func (a *API) Withdraw(c *gin.Context) {
	var body struct {
		Amount int64 `json:"amount" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	err := a.wallet.Withdraw(c.GetInt64("userID"), body.Amount)
	switch {
	case errors.Is(err, service.ErrInsufficient):
		c.JSON(http.StatusConflict, gin.H{"error": "insufficient balance"})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "withdraw failed"})
	default:
		c.Status(http.StatusNoContent)
	}
}

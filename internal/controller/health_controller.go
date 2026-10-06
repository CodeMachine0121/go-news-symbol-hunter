package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthController struct{}

func NewHealthController() *HealthController {
	return &HealthController{}
}

func (healthController *HealthController) GetHealth(context *gin.Context) {
	context.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorResponseRule struct {
	Err    error
	Status int
	Code   string
}

type ErrorResponseTable []ErrorResponseRule

func (errorResponseTable ErrorResponseTable) Respond(context *gin.Context, err error) {
	for _, errorResponseRule := range errorResponseTable {
		if errors.Is(err, errorResponseRule.Err) {
			context.JSON(errorResponseRule.Status, ErrorResponseBody{Error: ErrorDetail{Code: errorResponseRule.Code, Message: errorResponseRule.Err.Error()}})
			return
		}
	}
	context.JSON(http.StatusServiceUnavailable, ErrorResponseBody{Error: ErrorDetail{Code: "service_unavailable", Message: "服務暫時無法使用"}})
}

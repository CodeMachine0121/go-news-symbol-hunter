package controller

import (
	"errors"
	"net/http"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/service"
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
			errorDetail := ErrorDetail{Code: errorResponseRule.Code, Message: errorResponseRule.Err.Error()}
			if symbolAmbiguousError, isAmbiguous := errors.AsType[*service.SymbolAmbiguousError](err); isAmbiguous {
				errorDetail.Candidates = symbolAmbiguousError.Candidates
			}
			context.JSON(errorResponseRule.Status, ErrorResponseBody{Error: errorDetail})
			return
		}
	}
	context.JSON(http.StatusServiceUnavailable, ErrorResponseBody{Error: ErrorDetail{Code: "service_unavailable", Message: "服務暫時無法使用"}})
}

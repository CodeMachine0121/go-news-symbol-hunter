package controller

import (
	"net/http"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/application"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/service"
	"github.com/gin-gonic/gin"
)

var trackedSymbolGradeErrorResponseTable = ErrorResponseTable{
	{Err: service.ErrSymbolAndCategoryRequiredTogether, Status: http.StatusBadRequest, Code: "symbol_and_category_required_together"},
	{Err: service.ErrMarketCategoryUnsupported, Status: http.StatusBadRequest, Code: "market_category_unsupported"},
	{Err: service.ErrSymbolRequired, Status: http.StatusBadRequest, Code: "symbol_required"},
}

type TrackedSymbolGradeController struct {
	sessionGradeApplication *application.SessionGradeApplication
}

func NewTrackedSymbolGradeController(sessionGradeApplication *application.SessionGradeApplication) *TrackedSymbolGradeController {
	return &TrackedSymbolGradeController{sessionGradeApplication: sessionGradeApplication}
}

func (trackedSymbolGradeController *TrackedSymbolGradeController) GetTrackedSymbolGrades(context *gin.Context) {
	trackedSymbolGrades, err := trackedSymbolGradeController.sessionGradeApplication.GetTrackedSymbolGrades(context.Request.Context(), dto.GetTrackedSymbolGradesDto{
		Symbol:   context.Query("symbol"),
		Category: context.Query("category"),
	})
	if err != nil {
		trackedSymbolGradeErrorResponseTable.Respond(context, err)
		return
	}
	context.JSON(http.StatusOK, trackedSymbolGrades)
}

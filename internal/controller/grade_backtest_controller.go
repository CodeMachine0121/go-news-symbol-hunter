package controller

import (
	"net/http"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/application"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/service"
	"github.com/gin-gonic/gin"
)

var gradeBacktestErrorResponseTable = ErrorResponseTable{
	{Err: service.ErrBacktestPeriodInvalid, Status: http.StatusBadRequest, Code: "backtest_period_invalid"},
	{Err: service.ErrBacktestAnalysisLimitExceeded, Status: http.StatusBadRequest, Code: "backtest_analysis_limit_exceeded"},
	{Err: service.ErrSymbolAndCategoryRequiredTogether, Status: http.StatusBadRequest, Code: "symbol_and_category_required_together"},
	{Err: service.ErrMarketCategoryUnsupported, Status: http.StatusBadRequest, Code: "market_category_unsupported"},
	{Err: service.ErrSymbolRequired, Status: http.StatusBadRequest, Code: "symbol_required"},
	{Err: service.ErrPriceServiceUnavailable, Status: http.StatusServiceUnavailable, Code: "price_service_unavailable"},
}

type GradeBacktestController struct {
	gradeBacktestApplication *application.GradeBacktestApplication
}

func NewGradeBacktestController(gradeBacktestApplication *application.GradeBacktestApplication) *GradeBacktestController {
	return &GradeBacktestController{gradeBacktestApplication: gradeBacktestApplication}
}

func (gradeBacktestController *GradeBacktestController) GetGradeBacktest(context *gin.Context) {
	gradeBacktest, err := gradeBacktestController.gradeBacktestApplication.GetGradeBacktest(context.Request.Context(), dto.GetGradeBacktestDto{
		CompletedFrom: context.Query("completedFrom"),
		CompletedTo:   context.Query("completedTo"),
		Symbol:        context.Query("symbol"),
		Category:      context.Query("category"),
	})
	if err != nil {
		gradeBacktestErrorResponseTable.Respond(context, err)
		return
	}
	context.JSON(http.StatusOK, gradeBacktest)
}

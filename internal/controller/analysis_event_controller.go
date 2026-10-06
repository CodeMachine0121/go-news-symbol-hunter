package controller

import (
	"errors"
	"io"
	"net/http"
	"slices"
	"strconv"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/application"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/service"
	"github.com/gin-gonic/gin"
)

var analysisEventErrorResponseTable = ErrorResponseTable(slices.Concat(symbolErrorResponseRules, []ErrorResponseRule{
	{Err: service.ErrAnalysisEventNotFound, Status: http.StatusNotFound, Code: "analysis_event_not_found"},
}))

type StartSymbolAnalysisRequest struct {
	Symbol   string `json:"symbol"`
	Category string `json:"category"`
}

type AnalysisEventController struct {
	symbolAnalysisApplication *application.SymbolAnalysisApplication
}

func NewAnalysisEventController(symbolAnalysisApplication *application.SymbolAnalysisApplication) *AnalysisEventController {
	return &AnalysisEventController{symbolAnalysisApplication: symbolAnalysisApplication}
}

func (analysisEventController *AnalysisEventController) StartSymbolAnalysis(context *gin.Context) {
	var startSymbolAnalysisRequest StartSymbolAnalysisRequest
	if err := context.ShouldBindJSON(&startSymbolAnalysisRequest); err != nil && !errors.Is(err, io.EOF) {
		context.JSON(http.StatusBadRequest, ErrorResponseBody{Error: ErrorDetail{Code: "invalid_request_body", Message: "請求內容格式錯誤"}})
		return
	}
	startedSymbolAnalysis, err := analysisEventController.symbolAnalysisApplication.StartSymbolAnalysis(context.Request.Context(), dto.StartSymbolAnalysisDto{
		ApiKeyID: context.GetUint(AuthorizedApiKeyIDContextKey),
		Symbol:   startSymbolAnalysisRequest.Symbol,
		Category: startSymbolAnalysisRequest.Category,
	})
	if err != nil {
		analysisEventErrorResponseTable.Respond(context, err)
		return
	}
	responseStatus := http.StatusOK
	if startedSymbolAnalysis.IsNew {
		responseStatus = http.StatusAccepted
	}
	context.JSON(responseStatus, startedSymbolAnalysis.AnalysisEvent)
}

func (analysisEventController *AnalysisEventController) GetAnalysisEvent(context *gin.Context) {
	analysisEventID, err := strconv.ParseUint(context.Param("analysisEventId"), 10, 32)
	if err != nil {
		analysisEventErrorResponseTable.Respond(context, service.ErrAnalysisEventNotFound)
		return
	}
	analysisEvent, err := analysisEventController.symbolAnalysisApplication.GetAnalysisEvent(context.Request.Context(), uint(analysisEventID))
	if err != nil {
		analysisEventErrorResponseTable.Respond(context, err)
		return
	}
	context.JSON(http.StatusOK, analysisEvent)
}

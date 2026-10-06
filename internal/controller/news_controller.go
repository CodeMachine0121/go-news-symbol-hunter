package controller

import (
	"net/http"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/application"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/service"
	"github.com/gin-gonic/gin"
)

var symbolErrorResponseRules = []ErrorResponseRule{
	{Err: service.ErrMarketCategoryUnsupported, Status: http.StatusBadRequest, Code: "market_category_unsupported"},
	{Err: service.ErrSymbolRequired, Status: http.StatusBadRequest, Code: "symbol_required"},
	{Err: service.ErrSymbolNotFound, Status: http.StatusNotFound, Code: "symbol_not_found"},
	{Err: service.ErrNewsProvidersUnavailable, Status: http.StatusBadGateway, Code: "news_providers_unavailable"},
}

var newsErrorResponseTable = ErrorResponseTable(symbolErrorResponseRules)

type NewsController struct {
	newsSearchApplication *application.NewsSearchApplication
}

func NewNewsController(newsSearchApplication *application.NewsSearchApplication) *NewsController {
	return &NewsController{newsSearchApplication: newsSearchApplication}
}

func (newsController *NewsController) SearchSymbolNews(context *gin.Context) {
	symbolNews, err := newsController.newsSearchApplication.SearchSymbolNews(context.Request.Context(), dto.SearchSymbolNewsDto{
		Symbol:   context.Query("symbol"),
		Category: context.Query("category"),
	})
	if err != nil {
		newsErrorResponseTable.Respond(context, err)
		return
	}
	context.JSON(http.StatusOK, symbolNews)
}

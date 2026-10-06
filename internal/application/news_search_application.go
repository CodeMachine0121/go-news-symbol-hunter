package application

import (
	"context"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/service"
)

type NewsSearchApplication struct {
	newsSearchService *service.NewsSearchService
}

func NewNewsSearchApplication(newsSearchService *service.NewsSearchService) *NewsSearchApplication {
	return &NewsSearchApplication{newsSearchService: newsSearchService}
}

func (newsSearchApplication *NewsSearchApplication) SearchSymbolNews(ctx context.Context, searchSymbolNewsDto dto.SearchSymbolNewsDto) (dto.SymbolNewsDto, error) {
	return newsSearchApplication.newsSearchService.SearchSymbolNews(ctx, searchSymbolNewsDto)
}

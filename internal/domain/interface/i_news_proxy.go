package interfaces

import (
	"context"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

type INewsProxy interface {
	ProviderName() string
	FetchNews(ctx context.Context, searchKeyword string) ([]vo.NewsVo, error)
}

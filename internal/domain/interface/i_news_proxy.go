package interfaces

import "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"

type INewsProxy interface {
	ProviderName() string
	FetchNews(searchKeyword string) ([]vo.NewsVo, error)
}

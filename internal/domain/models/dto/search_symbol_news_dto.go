package dto

import "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"

type SearchSymbolNewsDto struct {
	Symbol   string
	Category string
	// a zero window keeps every news item inside the news search window
	NewsPublishedWindow vo.NewsPublishedWindowVo
}

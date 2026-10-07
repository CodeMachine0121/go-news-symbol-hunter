package dto

import "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"

type ConsultAnalystDto struct {
	Symbol        string
	Category      string
	SearchKeyword string
	// a zero window lets the analyst read every news item the news search keeps
	NewsPublishedWindow vo.NewsPublishedWindowVo
}

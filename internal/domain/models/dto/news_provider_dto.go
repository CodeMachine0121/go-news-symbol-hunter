package dto

import interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"

type NewsProviderDto struct {
	NewsProxy               interfaces.INewsProxy
	RequiresRelevanceFilter bool
}

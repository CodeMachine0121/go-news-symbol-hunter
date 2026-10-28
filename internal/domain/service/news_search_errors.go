package service

import (
	"errors"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

var (
	ErrMarketCategoryUnsupported = vo.ErrMarketCategoryUnsupported
	ErrSymbolRequired            = vo.ErrSymbolRequired
	ErrSymbolNotFound            = domains.ErrSymbolNotFound
	ErrSymbolAmbiguous           = domains.ErrSymbolAmbiguous
	ErrNewsProvidersUnavailable  = errors.New("新聞來源暫時無法使用")
)

type SymbolAmbiguousError = domains.SymbolAmbiguousError

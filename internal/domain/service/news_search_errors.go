package service

import (
	"errors"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

var (
	ErrMarketCategoryUnsupported = vo.ErrMarketCategoryUnsupported
	ErrSymbolRequired            = vo.ErrSymbolRequired
	ErrSymbolNotFound            = errors.New("找不到此標的")
	ErrNewsProvidersUnavailable  = errors.New("新聞來源暫時無法使用")
)

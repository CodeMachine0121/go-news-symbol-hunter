package interfaces

import (
	"context"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

// an unknown symbol yields no candles rather than an error; an error means the source could not answer
type IPriceHistoryProxy interface {
	FetchCandles(ctx context.Context, priceHistoryQuery vo.PriceHistoryQueryVo) ([]vo.PriceCandleVo, error)
}

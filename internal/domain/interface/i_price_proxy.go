package interfaces

import (
	"context"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

type IPriceProxy interface {
	FetchPrice(ctx context.Context, symbol string) (vo.PriceQuoteVo, error)
}

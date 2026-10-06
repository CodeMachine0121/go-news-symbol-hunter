package service

import (
	"context"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

type PriceSnapshotService struct {
	priceProviderCatalog dto.PriceProviderCatalogDto
}

func NewPriceSnapshotService(priceProviderCatalog dto.PriceProviderCatalogDto) *PriceSnapshotService {
	return &PriceSnapshotService{priceProviderCatalog: priceProviderCatalog}
}

// never fails: an analysis is still worth keeping without its price
func (priceSnapshotService *PriceSnapshotService) CapturePrice(ctx context.Context, capturePriceDto dto.CapturePriceDto) *vo.PriceQuoteVo {
	priceProxy := priceSnapshotService.priceProviderCatalog.UsStock
	switch capturePriceDto.Category {
	case vo.MarketCategoryTwStock:
		priceProxy = priceSnapshotService.priceProviderCatalog.TwStock
	case vo.MarketCategoryCrypto:
		priceProxy = priceSnapshotService.priceProviderCatalog.Crypto
	}
	priceQuote, err := priceProxy.FetchPrice(ctx, capturePriceDto.Symbol)
	if err != nil {
		return nil
	}
	return &priceQuote
}

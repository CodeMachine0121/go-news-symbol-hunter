package service

import (
	"context"
	"time"

	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

const PriceCaptureTimeout = 10 * time.Second

type PriceSnapshotService struct {
	priceProviderCatalog dto.PriceProviderCatalogDto
}

func NewPriceSnapshotService(priceProviderCatalog dto.PriceProviderCatalogDto) *PriceSnapshotService {
	return &PriceSnapshotService{priceProviderCatalog: priceProviderCatalog}
}

// never fails: an analysis is still worth keeping without its price
func (priceSnapshotService *PriceSnapshotService) CapturePrice(ctx context.Context, capturePriceDto dto.CapturePriceDto) *vo.PriceQuoteVo {
	priceProxiesByCategory := map[string]interfaces.IPriceProxy{
		vo.MarketCategoryTwStock: priceSnapshotService.priceProviderCatalog.TwStock,
		vo.MarketCategoryUsStock: priceSnapshotService.priceProviderCatalog.UsStock,
		vo.MarketCategoryCrypto:  priceSnapshotService.priceProviderCatalog.Crypto,
	}
	priceProxy, supported := priceProxiesByCategory[capturePriceDto.Category]
	if !supported {
		return nil
	}
	captureContext, cancelCapture := context.WithTimeout(ctx, PriceCaptureTimeout)
	defer cancelCapture()
	priceQuote, err := priceProxy.FetchPrice(captureContext, capturePriceDto.Symbol)
	if err != nil {
		return nil
	}
	return &priceQuote
}

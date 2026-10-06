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
	priceProxiesByCategory map[string]interfaces.IPriceProxy
}

func NewPriceSnapshotService(priceProviderCatalog dto.PriceProviderCatalogDto) *PriceSnapshotService {
	return &PriceSnapshotService{priceProxiesByCategory: map[string]interfaces.IPriceProxy{
		vo.MarketCategoryTwStock: priceProviderCatalog.TwStock,
		vo.MarketCategoryUsStock: priceProviderCatalog.UsStock,
		vo.MarketCategoryCrypto:  priceProviderCatalog.Crypto,
	}}
}

// never fails: an analysis is still worth keeping without its price
func (priceSnapshotService *PriceSnapshotService) CapturePrice(ctx context.Context, capturePriceDto dto.CapturePriceDto) *vo.PriceQuoteVo {
	priceProxy, supported := priceSnapshotService.priceProxiesByCategory[capturePriceDto.Category]
	if !supported {
		return nil
	}
	// its own budget, independent of how much of the analysis deadline is left
	captureContext, cancelCapture := context.WithTimeout(context.WithoutCancel(ctx), PriceCaptureTimeout)
	defer cancelCapture()
	priceQuote, err := priceProxy.FetchPrice(captureContext, capturePriceDto.Symbol)
	if err != nil {
		return nil
	}
	return &priceQuote
}

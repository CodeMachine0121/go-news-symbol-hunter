package service

import (
	"context"
	"sync"
	"time"

	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

const PriceCaptureTimeout = 10 * time.Second

type PriceSnapshotService struct {
	priceProxiesByCategory map[string][]interfaces.IPriceProxy
}

func NewPriceSnapshotService(priceProviderCatalog dto.PriceProviderCatalogDto) *PriceSnapshotService {
	return &PriceSnapshotService{priceProxiesByCategory: map[string][]interfaces.IPriceProxy{
		vo.MarketCategoryTwStock: priceProviderCatalog.TwStock,
		vo.MarketCategoryUsStock: priceProviderCatalog.UsStock,
		vo.MarketCategoryCrypto:  priceProviderCatalog.Crypto,
	}}
}

// never fails: an analysis is still worth keeping without its price
func (priceSnapshotService *PriceSnapshotService) CapturePrice(ctx context.Context, capturePriceDto dto.CapturePriceDto) *vo.PriceQuoteVo {
	priceProxies, supported := priceSnapshotService.priceProxiesByCategory[capturePriceDto.Category]
	if !supported {
		return nil
	}
	// its own budget, independent of how much of the analysis deadline is left
	captureContext, cancelCapture := context.WithTimeout(context.WithoutCancel(ctx), PriceCaptureTimeout)
	defer cancelCapture()
	// asked together so one slow source cannot use up the others' share of the budget
	priceQuotes := make([]vo.PriceQuoteVo, len(priceProxies))
	fetchErrors := make([]error, len(priceProxies))
	var waitGroup sync.WaitGroup
	for index, priceProxy := range priceProxies {
		waitGroup.Go(func() {
			priceQuotes[index], fetchErrors[index] = priceProxy.FetchPrice(captureContext, capturePriceDto.Symbol)
		})
	}
	waitGroup.Wait()
	for index := range priceProxies {
		if fetchErrors[index] == nil {
			return &priceQuotes[index]
		}
	}
	return nil
}

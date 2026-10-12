package service

import (
	"context"
	"fmt"

	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"golang.org/x/sync/errgroup"
)

const (
	MaximumBacktestAnalysisCount         = 500
	maximumConcurrentPriceHistoryFetches = 8
)

type GradeBacktestService struct {
	analysisResultRepository      interfaces.IAnalysisResultRepository
	clockProxy                    interfaces.IClockProxy
	priceHistoryProxiesByCategory map[string]interfaces.IPriceHistoryProxy
}

func NewGradeBacktestService(analysisResultRepository interfaces.IAnalysisResultRepository, clockProxy interfaces.IClockProxy, priceHistoryCatalog dto.PriceHistoryCatalogDto) *GradeBacktestService {
	return &GradeBacktestService{
		analysisResultRepository: analysisResultRepository,
		clockProxy:               clockProxy,
		priceHistoryProxiesByCategory: map[string]interfaces.IPriceHistoryProxy{
			vo.MarketCategoryTwStock: priceHistoryCatalog.TwStock,
			vo.MarketCategoryUsStock: priceHistoryCatalog.UsStock,
			vo.MarketCategoryCrypto:  priceHistoryCatalog.Crypto,
		},
	}
}

func (gradeBacktestService *GradeBacktestService) GetGradeBacktest(ctx context.Context, getGradeBacktestDto dto.GetGradeBacktestDto) (dto.GradeBacktestDto, error) {
	completedPeriod, err := vo.NewAnalysisCompletedPeriodVo(getGradeBacktestDto.CompletedFrom, getGradeBacktestDto.CompletedTo)
	if err != nil {
		return dto.GradeBacktestDto{}, err
	}
	symbolFilter, err := vo.NewSymbolFilterVo(getGradeBacktestDto.Symbol, getGradeBacktestDto.Category)
	if err != nil {
		return dto.GradeBacktestDto{}, err
	}
	gradeBacktestFilter := vo.GradeBacktestFilterVo{CompletedPeriod: completedPeriod, SymbolFilterVo: symbolFilter}
	// one past the limit tells "exactly at the limit" apart from "over it"
	analysisResults, err := gradeBacktestService.analysisResultRepository.FindForBacktest(ctx, gradeBacktestFilter, MaximumBacktestAnalysisCount+1)
	if err != nil {
		return dto.GradeBacktestDto{}, fmt.Errorf("%w: %v", ErrBacktestStorageUnavailable, err)
	}
	if len(analysisResults) > MaximumBacktestAnalysisCount {
		return dto.GradeBacktestDto{}, ErrBacktestAnalysisLimitExceeded
	}
	evaluatedAt := gradeBacktestService.clockProxy.Now()
	analysisBacktests := make([]dto.AnalysisBacktestDto, len(analysisResults))
	priceFetchGroup, priceFetchContext := errgroup.WithContext(ctx)
	priceFetchGroup.SetLimit(maximumConcurrentPriceHistoryFetches)
	for index, analysisResult := range analysisResults {
		analysisBacktest := domains.NewAnalysisBacktestDomain(analysisResult)
		priceHistoryProxy := gradeBacktestService.priceHistoryProxiesByCategory[analysisResult.Category]
		if priceHistoryProxy == nil {
			analysisBacktests[index] = analysisBacktest.Unevaluable()
			continue
		}
		priceFetchGroup.Go(func() error {
			priceWindows := analysisBacktest.PriceWindows()
			minuteCandles, err := priceHistoryProxy.FetchCandles(priceFetchContext, priceWindows.Minute)
			if err != nil {
				return err
			}
			hourlyCandles := []vo.PriceCandleVo{}
			if analysisBacktest.NeedsHourlyFallback(minuteCandles) {
				hourlyCandles, err = priceHistoryProxy.FetchCandles(priceFetchContext, priceWindows.HourlyFallback)
				if err != nil {
					return err
				}
			}
			dailyCandles, err := priceHistoryProxy.FetchCandles(priceFetchContext, priceWindows.Daily)
			if err != nil {
				return err
			}
			analysisBacktests[index] = analysisBacktest.Evaluate(vo.BacktestPricesVo{MinuteCandles: minuteCandles, HourlyCandles: hourlyCandles, DailyCandles: dailyCandles}, evaluatedAt)
			return nil
		})
	}
	// one unanswered request means every other answer is suspect too, so the whole backtest fails
	// rather than reporting analyses as unevaluable
	if err := priceFetchGroup.Wait(); err != nil {
		return dto.GradeBacktestDto{}, fmt.Errorf("%w: %v", ErrPriceServiceUnavailable, err)
	}
	return domains.NewGradeBacktestSummaryDomain(analysisBacktests).ToDto(), nil
}

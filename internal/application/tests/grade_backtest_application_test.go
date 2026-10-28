package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/application"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface/mocks"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/service"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Wednesday 2026-10-07 10:15 in Taipei
var backtestCompletedAt = time.Date(2026, 10, 7, 10, 15, 0, 0, taipei)

var errPriceServiceDown = errors.New("connection refused")

// long after every checkpoint used below, so all of their days have closed
var backtestEvaluatedAt = time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC)

func clockAt(t *testing.T, now time.Time) *mocks.MockIClockProxy {
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(now).Maybe()
	return clockProxy
}

type gradeBacktestFixture struct {
	gradeBacktestApplication *application.GradeBacktestApplication
	gradeRecordRepository    *mocks.MockIGradeRecordRepository
	priceHistoryProxy        *mocks.MockIPriceHistoryProxy
}

func createGradeBacktestFixture(t *testing.T) gradeBacktestFixture {
	fixture := gradeBacktestFixture{
		gradeRecordRepository: mocks.NewMockIGradeRecordRepository(t),
		priceHistoryProxy:     mocks.NewMockIPriceHistoryProxy(t),
	}
	fixture.gradeBacktestApplication = application.NewGradeBacktestApplication(service.NewGradeBacktestService(
		fixture.gradeRecordRepository,
		clockAt(t, backtestEvaluatedAt),
		dto.PriceHistoryCatalogDto{TwStock: fixture.priceHistoryProxy, Crypto: fixture.priceHistoryProxy},
	))
	return fixture
}

func completedAnalysis(analysisEventID uint, symbol string, category string, grade string) entities.GradeRecord {
	return entities.GradeRecord{Source: "manual", AnalysisEventID: new(uint(analysisEventID)), Symbol: symbol, Category: category, Grade: grade, Confidence: 80, TimeHorizon: "short", CreatedAt: backtestCompletedAt}
}

func priceCandle(openTime time.Time, close string) vo.PriceCandleVo {
	return vo.PriceCandleVo{OpenTime: openTime, Close: decimal.RequireFromString(close)}
}

// ten trading days from Thursday 2026-10-08, the last one closing at lastClose
func tenTradingDaysEndingAt(lastClose string) []vo.PriceCandleVo {
	dailyCandles := []vo.PriceCandleVo{}
	for day := range 10 {
		close := "100"
		if day == 9 {
			close = lastClose
		}
		dailyCandles = append(dailyCandles, priceCandle(time.Date(2026, 10, 8+day, 0, 0, 0, 0, time.UTC), close))
	}
	return dailyCandles
}

func (fixture gradeBacktestFixture) givenAnalyses(analysisResults ...entities.GradeRecord) {
	fixture.gradeRecordRepository.EXPECT().FindForBacktest(mock.Anything, mock.Anything, 501).Return(analysisResults, nil).Once()
}

func (fixture gradeBacktestFixture) givenCandles(symbol string, interval string, candles []vo.PriceCandleVo) {
	fixture.priceHistoryProxy.EXPECT().FetchCandles(mock.Anything, mock.MatchedBy(func(priceHistoryQuery vo.PriceHistoryQueryVo) bool {
		return priceHistoryQuery.Symbol == symbol && priceHistoryQuery.Interval == interval
	})).Return(candles, nil)
}

func (fixture gradeBacktestFixture) givenTradedAt(symbol string, entryClose string, checkpointClose string) {
	fixture.givenCandles(symbol, "1m", []vo.PriceCandleVo{priceCandle(backtestCompletedAt, entryClose)})
	fixture.givenCandles(symbol, "1d", tenTradingDaysEndingAt(checkpointClose))
}

func TestGetGradeBacktest_JudgesEachAnalysisAndSummarises(t *testing.T) {
	fixture := createGradeBacktestFixture(t)
	fixture.givenAnalyses(completedAnalysis(1, "2330", "twStock", "bullish"), completedAnalysis(2, "BTC", "crypto", "bearish"))
	fixture.givenTradedAt("2330", "100", "105")
	fixture.givenTradedAt("BTC", "100", "103")

	gradeBacktest, err := fixture.gradeBacktestApplication.GetGradeBacktest(context.Background(), dto.GetGradeBacktestDto{})

	require.NoError(t, err)
	require.Len(t, gradeBacktest.Analyses, 2)
	assert.Equal(t, "hit", gradeBacktest.Analyses[0].Outcome)
	assert.Equal(t, "miss", gradeBacktest.Analyses[1].Outcome)
	assert.Equal(t, 0.5, *gradeBacktest.Overall.HitRate)
	assert.Equal(t, 0.5, gradeBacktest.RandomBaseline)
}

func TestGetGradeBacktest_PassesThePeriodAndSymbolOn(t *testing.T) {
	testCases := []struct {
		name             string
		getGradeBacktest dto.GetGradeBacktestDto
		expectedFrom     time.Time
		expectedBefore   time.Time
		expectedSymbol   string
		expectedCategory string
		expectedSource   string
	}{
		{name: "no conditions covers everything"},
		{name: "only background records", getGradeBacktest: dto.GetGradeBacktestDto{Source: "background"}, expectedSource: "background"},
		{name: "only manual records", getGradeBacktest: dto.GetGradeBacktestDto{Source: "manual"}, expectedSource: "manual"},
		{
			name:             "october in taipei",
			getGradeBacktest: dto.GetGradeBacktestDto{CompletedFrom: "2026-10-01", CompletedTo: "2026-10-31"},
			expectedFrom:     time.Date(2026, 10, 1, 0, 0, 0, 0, taipei),
			expectedBefore:   time.Date(2026, 11, 1, 0, 0, 0, 0, taipei),
		},
		{name: "a symbol is written the way analyses store it", getGradeBacktest: dto.GetGradeBacktestDto{Symbol: " btc ", Category: "crypto"}, expectedSymbol: "BTC", expectedCategory: "crypto"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := createGradeBacktestFixture(t)
			receivedFilter := vo.GradeBacktestFilterVo{}
			fixture.gradeRecordRepository.EXPECT().FindForBacktest(mock.Anything, mock.Anything, 501).RunAndReturn(func(_ context.Context, gradeBacktestFilter vo.GradeBacktestFilterVo, _ int) ([]entities.GradeRecord, error) {
				receivedFilter = gradeBacktestFilter
				return []entities.GradeRecord{}, nil
			})

			gradeBacktest, err := fixture.gradeBacktestApplication.GetGradeBacktest(context.Background(), testCase.getGradeBacktest)

			require.NoError(t, err)
			assert.Empty(t, gradeBacktest.Analyses)
			assert.True(t, testCase.expectedFrom.Equal(receivedFilter.CompletedPeriod.From))
			assert.True(t, testCase.expectedBefore.Equal(receivedFilter.CompletedPeriod.Before))
			assert.Equal(t, testCase.expectedSymbol, receivedFilter.Symbol)
			assert.Equal(t, testCase.expectedCategory, receivedFilter.Category)
			assert.Equal(t, testCase.expectedSource, receivedFilter.Source)
		})
	}
}

func TestGetGradeBacktest_RejectsAnUnusableRequest(t *testing.T) {
	testCases := []struct {
		name             string
		getGradeBacktest dto.GetGradeBacktestDto
		expectedError    error
	}{
		{name: "symbol without category", getGradeBacktest: dto.GetGradeBacktestDto{Symbol: "2330"}, expectedError: service.ErrSymbolAndCategoryRequiredTogether},
		{name: "category without symbol", getGradeBacktest: dto.GetGradeBacktestDto{Category: "twStock"}, expectedError: service.ErrSymbolAndCategoryRequiredTogether},
		{name: "unsupported category", getGradeBacktest: dto.GetGradeBacktestDto{Symbol: "2330", Category: "futures"}, expectedError: service.ErrMarketCategoryUnsupported},
		{name: "blank symbol", getGradeBacktest: dto.GetGradeBacktestDto{Symbol: " ", Category: "twStock"}, expectedError: service.ErrSymbolRequired},
		{name: "start after end", getGradeBacktest: dto.GetGradeBacktestDto{CompletedFrom: "2026-10-31", CompletedTo: "2026-10-01"}, expectedError: service.ErrBacktestPeriodInvalid},
		{name: "unreadable date", getGradeBacktest: dto.GetGradeBacktestDto{CompletedFrom: "10/01"}, expectedError: service.ErrBacktestPeriodInvalid},
		{name: "unknown source", getGradeBacktest: dto.GetGradeBacktestDto{Source: "other"}, expectedError: service.ErrGradeSourceInvalid},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := createGradeBacktestFixture(t)

			_, err := fixture.gradeBacktestApplication.GetGradeBacktest(context.Background(), testCase.getGradeBacktest)

			assert.ErrorIs(t, err, testCase.expectedError)
		})
	}
}

func TestGetGradeBacktest_CoversAtMostFiveHundredAnalyses(t *testing.T) {
	testCases := []struct {
		name          string
		analysisCount int
		expectedError error
	}{
		{name: "exactly five hundred", analysisCount: 500},
		{name: "five hundred and one", analysisCount: 501, expectedError: service.ErrBacktestAnalysisLimitExceeded},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := createGradeBacktestFixture(t)
			analysisResults := []entities.GradeRecord{}
			for analysisEventID := range testCase.analysisCount {
				analysisResults = append(analysisResults, completedAnalysis(uint(analysisEventID), "AAPL", "usStock", "bullish"))
			}
			fixture.givenAnalyses(analysisResults...)

			gradeBacktest, err := fixture.gradeBacktestApplication.GetGradeBacktest(context.Background(), dto.GetGradeBacktestDto{})

			if testCase.expectedError != nil {
				assert.ErrorIs(t, err, testCase.expectedError)
				return
			}
			require.NoError(t, err)
			assert.Len(t, gradeBacktest.Analyses, testCase.analysisCount)
		})
	}
}

func TestGetGradeBacktest_UsStockIsUnevaluableWithoutAskingForPrices(t *testing.T) {
	fixture := createGradeBacktestFixture(t)
	fixture.givenAnalyses(completedAnalysis(1, "AAPL", "usStock", "bullish"))

	gradeBacktest, err := fixture.gradeBacktestApplication.GetGradeBacktest(context.Background(), dto.GetGradeBacktestDto{})

	require.NoError(t, err)
	assert.Equal(t, "unevaluable", gradeBacktest.Analyses[0].Outcome)
	assert.Equal(t, 1, gradeBacktest.OutcomeCounts["unevaluable"])
	fixture.priceHistoryProxy.AssertNotCalled(t, "FetchCandles", mock.Anything, mock.Anything)
}

func TestGetGradeBacktest_ASymbolThePriceServiceDoesNotKnowIsUnevaluable(t *testing.T) {
	fixture := createGradeBacktestFixture(t)
	fixture.givenAnalyses(completedAnalysis(1, "6488", "twStock", "bullish"))
	fixture.givenCandles("6488", "1m", []vo.PriceCandleVo{})
	fixture.givenCandles("6488", "1h", []vo.PriceCandleVo{})
	fixture.givenCandles("6488", "1d", []vo.PriceCandleVo{})

	gradeBacktest, err := fixture.gradeBacktestApplication.GetGradeBacktest(context.Background(), dto.GetGradeBacktestDto{})

	require.NoError(t, err)
	assert.Equal(t, "unevaluable", gradeBacktest.Analyses[0].Outcome)
}

func TestGetGradeBacktest_FallsBackToHourlyCandlesOnlyWithoutRecentMinutes(t *testing.T) {
	fixture := createGradeBacktestFixture(t)
	fixture.givenAnalyses(completedAnalysis(1, "2330", "twStock", "bullish"))
	fixture.givenCandles("2330", "1m", []vo.PriceCandleVo{})
	fixture.givenCandles("2330", "1h", []vo.PriceCandleVo{priceCandle(backtestCompletedAt.Add(-time.Hour), "100")})
	fixture.givenCandles("2330", "1d", tenTradingDaysEndingAt("105"))

	gradeBacktest, err := fixture.gradeBacktestApplication.GetGradeBacktest(context.Background(), dto.GetGradeBacktestDto{})

	require.NoError(t, err)
	assert.Equal(t, "100", gradeBacktest.Analyses[0].EntryPrice.String())
	assert.Equal(t, "hit", gradeBacktest.Analyses[0].Outcome)
}

func TestGetGradeBacktest_DoesNotAskForHourlyCandlesWhenMinutesExist(t *testing.T) {
	fixture := createGradeBacktestFixture(t)
	fixture.givenAnalyses(completedAnalysis(1, "2330", "twStock", "bullish"))
	fixture.givenTradedAt("2330", "100", "105")

	_, err := fixture.gradeBacktestApplication.GetGradeBacktest(context.Background(), dto.GetGradeBacktestDto{})

	require.NoError(t, err)
	fixture.priceHistoryProxy.AssertNotCalled(t, "FetchCandles", mock.Anything, mock.MatchedBy(func(priceHistoryQuery vo.PriceHistoryQueryVo) bool {
		return priceHistoryQuery.Interval == "1h"
	}))
}

func TestGetGradeBacktest_FailsWholeWhenThePriceServiceDoesNotAnswer(t *testing.T) {
	testCases := []struct {
		name           string
		failedInterval string
	}{
		{name: "minute candles", failedInterval: "1m"},
		{name: "hourly candles", failedInterval: "1h"},
		{name: "daily candles", failedInterval: "1d"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := createGradeBacktestFixture(t)
			fixture.givenAnalyses(completedAnalysis(1, "2330", "twStock", "bullish"))
			fixture.priceHistoryProxy.EXPECT().FetchCandles(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, priceHistoryQuery vo.PriceHistoryQueryVo) ([]vo.PriceCandleVo, error) {
				if priceHistoryQuery.Interval == testCase.failedInterval {
					return nil, errPriceServiceDown
				}
				return []vo.PriceCandleVo{}, nil
			}).Maybe()

			_, err := fixture.gradeBacktestApplication.GetGradeBacktest(context.Background(), dto.GetGradeBacktestDto{})

			assert.ErrorIs(t, err, service.ErrPriceServiceUnavailable)
		})
	}
}

func TestGetGradeBacktest_FailsWhenTheAnalysesCannotBeRead(t *testing.T) {
	fixture := createGradeBacktestFixture(t)
	fixture.gradeRecordRepository.EXPECT().FindForBacktest(mock.Anything, mock.Anything, 501).Return(nil, errors.New("database down"))

	_, err := fixture.gradeBacktestApplication.GetGradeBacktest(context.Background(), dto.GetGradeBacktestDto{})

	assert.ErrorIs(t, err, service.ErrBacktestStorageUnavailable)
}

func TestGetGradeBacktest_AsksEachMarketsOwnPriceSource(t *testing.T) {
	gradeRecordRepository := mocks.NewMockIGradeRecordRepository(t)
	priceHistoryProxiesByCategory := map[string]*mocks.MockIPriceHistoryProxy{"twStock": mocks.NewMockIPriceHistoryProxy(t), "usStock": mocks.NewMockIPriceHistoryProxy(t), "crypto": mocks.NewMockIPriceHistoryProxy(t)}
	gradeBacktestApplication := application.NewGradeBacktestApplication(service.NewGradeBacktestService(gradeRecordRepository, clockAt(t, backtestEvaluatedAt), dto.PriceHistoryCatalogDto{
		TwStock: priceHistoryProxiesByCategory["twStock"],
		UsStock: priceHistoryProxiesByCategory["usStock"],
		Crypto:  priceHistoryProxiesByCategory["crypto"],
	}))
	gradeRecordRepository.EXPECT().FindForBacktest(mock.Anything, mock.Anything, 501).Return([]entities.GradeRecord{
		completedAnalysis(1, "2330", "twStock", "bullish"), completedAnalysis(2, "AAPL", "usStock", "bullish"), completedAnalysis(3, "BTC", "crypto", "bullish"),
	}, nil)
	for category, priceHistoryProxy := range priceHistoryProxiesByCategory {
		priceHistoryProxy.EXPECT().FetchCandles(mock.Anything, mock.MatchedBy(func(priceHistoryQuery vo.PriceHistoryQueryVo) bool {
			return priceHistoryQuery.Category == category && priceHistoryQuery.Interval == "1m"
		})).Return([]vo.PriceCandleVo{priceCandle(backtestCompletedAt, "100")}, nil)
		priceHistoryProxy.EXPECT().FetchCandles(mock.Anything, mock.MatchedBy(func(priceHistoryQuery vo.PriceHistoryQueryVo) bool {
			return priceHistoryQuery.Category == category && priceHistoryQuery.Interval == "1d"
		})).Return(tenTradingDaysEndingAt("105"), nil)
	}

	gradeBacktest, err := gradeBacktestApplication.GetGradeBacktest(context.Background(), dto.GetGradeBacktestDto{})

	require.NoError(t, err)
	assert.Equal(t, map[string]int{"hit": 3, "miss": 0, "neutral": 0, "pending": 0, "unevaluable": 0}, gradeBacktest.OutcomeCounts)
}

func TestGetGradeBacktest_ACheckpointDayStillTradingIsNotYetDue(t *testing.T) {
	gradeRecordRepository := mocks.NewMockIGradeRecordRepository(t)
	priceHistoryProxy := mocks.NewMockIPriceHistoryProxy(t)
	// the tenth trading day is 2026-10-17 and it is still trading
	duringTheCheckpointDay := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	gradeBacktestApplication := application.NewGradeBacktestApplication(service.NewGradeBacktestService(gradeRecordRepository, clockAt(t, duringTheCheckpointDay), dto.PriceHistoryCatalogDto{Crypto: priceHistoryProxy}))
	gradeRecordRepository.EXPECT().FindForBacktest(mock.Anything, mock.Anything, 501).Return([]entities.GradeRecord{completedAnalysis(1, "BTC", "crypto", "bullish")}, nil)
	priceHistoryProxy.EXPECT().FetchCandles(mock.Anything, mock.MatchedBy(func(priceHistoryQuery vo.PriceHistoryQueryVo) bool { return priceHistoryQuery.Interval == "1m" })).
		Return([]vo.PriceCandleVo{priceCandle(backtestCompletedAt, "100")}, nil)
	priceHistoryProxy.EXPECT().FetchCandles(mock.Anything, mock.MatchedBy(func(priceHistoryQuery vo.PriceHistoryQueryVo) bool { return priceHistoryQuery.Interval == "1d" })).
		Return(tenTradingDaysEndingAt("105"), nil)

	gradeBacktest, err := gradeBacktestApplication.GetGradeBacktest(context.Background(), dto.GetGradeBacktestDto{})

	require.NoError(t, err)
	assert.Equal(t, "pending", gradeBacktest.Analyses[0].Outcome)
}

func TestGetGradeBacktest_OneSymbolWithoutPricesLeavesTheOthersJudged(t *testing.T) {
	fixture := createGradeBacktestFixture(t)
	fixture.givenAnalyses(completedAnalysis(1, "6488", "twStock", "bullish"), completedAnalysis(2, "2330", "twStock", "bullish"))
	fixture.givenCandles("6488", "1m", []vo.PriceCandleVo{})
	fixture.givenCandles("6488", "1h", []vo.PriceCandleVo{})
	fixture.givenCandles("6488", "1d", []vo.PriceCandleVo{})
	fixture.givenTradedAt("2330", "100", "105")

	gradeBacktest, err := fixture.gradeBacktestApplication.GetGradeBacktest(context.Background(), dto.GetGradeBacktestDto{})

	require.NoError(t, err)
	assert.Equal(t, "unevaluable", gradeBacktest.Analyses[0].Outcome)
	assert.Equal(t, "hit", gradeBacktest.Analyses[1].Outcome)
	assert.Equal(t, 1.0, *gradeBacktest.Overall.HitRate)
}

func TestGetGradeBacktest_CryptoWithoutRecentMinutesIsUnevaluableWithoutAskingForHourlyCandles(t *testing.T) {
	fixture := createGradeBacktestFixture(t)
	fixture.givenAnalyses(completedAnalysis(1, "BTC", "crypto", "bullish"))
	fixture.givenCandles("BTC", "1m", []vo.PriceCandleVo{})
	fixture.givenCandles("BTC", "1d", tenTradingDaysEndingAt("105"))

	gradeBacktest, err := fixture.gradeBacktestApplication.GetGradeBacktest(context.Background(), dto.GetGradeBacktestDto{})

	require.NoError(t, err)
	assert.Equal(t, "unevaluable", gradeBacktest.Analyses[0].Outcome)
	fixture.priceHistoryProxy.AssertNotCalled(t, "FetchCandles", mock.Anything, mock.MatchedBy(func(priceHistoryQuery vo.PriceHistoryQueryVo) bool {
		return priceHistoryQuery.Interval == "1h"
	}))
}

func backgroundRecord(gradeRecordID uint, symbol string, grade string) entities.GradeRecord {
	return entities.GradeRecord{ID: gradeRecordID, Source: "background", Symbol: symbol, Category: "twStock", TradingDay: "2026-10-07", Grade: grade, Confidence: 55, TimeHorizon: "short", CreatedAt: backtestCompletedAt}
}

func TestGetGradeBacktest_ComparesManualAndBackgroundHitRates(t *testing.T) {
	fixture := createGradeBacktestFixture(t)
	gradeRecords := []entities.GradeRecord{}
	// manual: 6 of 10 hit; background: 9 of 20 hit
	for index := range 10 {
		symbol := "M-MISS"
		if index < 6 {
			symbol = "M-HIT"
		}
		gradeRecords = append(gradeRecords, entities.GradeRecord{Source: "manual", AnalysisEventID: new(uint(index + 1)), Symbol: symbol, Category: "twStock", Grade: "bullish", TimeHorizon: "short", CreatedAt: backtestCompletedAt})
	}
	for index := range 20 {
		symbol := "B-MISS"
		if index < 9 {
			symbol = "B-HIT"
		}
		gradeRecords = append(gradeRecords, backgroundRecord(uint(index+100), symbol, "bullish"))
	}
	fixture.givenAnalyses(gradeRecords...)
	fixture.givenTradedAt("M-HIT", "100", "105")
	fixture.givenTradedAt("M-MISS", "100", "95")
	fixture.givenTradedAt("B-HIT", "100", "105")
	fixture.givenTradedAt("B-MISS", "100", "95")

	gradeBacktest, err := fixture.gradeBacktestApplication.GetGradeBacktest(context.Background(), dto.GetGradeBacktestDto{})

	require.NoError(t, err)
	assert.Equal(t, 0.6, *gradeBacktest.BySource["manual"].HitRate)
	assert.Equal(t, 0.45, *gradeBacktest.BySource["background"].HitRate)
	assert.Equal(t, 0.5, *gradeBacktest.Overall.HitRate)
	assert.Equal(t, "background", gradeBacktest.Analyses[29].Source)
}

func TestGetGradeBacktest_OnlyBackgroundRecordsCountWhenTheyAreAllThereIs(t *testing.T) {
	fixture := createGradeBacktestFixture(t)
	gradeRecords := []entities.GradeRecord{}
	for index := range 20 {
		symbol := "B-MISS"
		if index < 9 {
			symbol = "B-HIT"
		}
		gradeRecords = append(gradeRecords, backgroundRecord(uint(index+100), symbol, "bullish"))
	}
	fixture.givenAnalyses(gradeRecords...)
	fixture.givenTradedAt("B-HIT", "100", "105")
	fixture.givenTradedAt("B-MISS", "100", "95")

	gradeBacktest, err := fixture.gradeBacktestApplication.GetGradeBacktest(context.Background(), dto.GetGradeBacktestDto{Source: "background"})

	require.NoError(t, err)
	assert.Equal(t, 0.45, *gradeBacktest.Overall.HitRate)
	assert.Nil(t, gradeBacktest.BySource["manual"].HitRate)
}

func TestGetGradeBacktest_NoBackgroundRecordsMeansNoDataNotZero(t *testing.T) {
	fixture := createGradeBacktestFixture(t)
	fixture.givenAnalyses(completedAnalysis(1, "2330", "twStock", "bullish"))
	fixture.givenTradedAt("2330", "100", "105")

	gradeBacktest, err := fixture.gradeBacktestApplication.GetGradeBacktest(context.Background(), dto.GetGradeBacktestDto{})

	require.NoError(t, err)
	assert.Equal(t, dto.HitRateDto{}, gradeBacktest.BySource["background"])
	assert.Equal(t, 1.0, *gradeBacktest.BySource["manual"].HitRate)
}

package domains_test

import (
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var backtestTaipei = time.FixedZone("Asia/Taipei", 8*60*60)

// long after every candle used below, so all of their days have closed
var longAfterwards = time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC)

// 2026-10-07 is a Wednesday
func taipeiAt(day int, hour int, minute int) time.Time {
	return time.Date(2026, 10, day, hour, minute, 0, 0, backtestTaipei)
}

func candleAt(openTime time.Time, close string) vo.PriceCandleVo {
	return vo.PriceCandleVo{OpenTime: openTime, Close: decimal.RequireFromString(close)}
}

// one daily candle per calendar day from firstDay (a UTC midnight), with the given closes
func dailyCandlesFrom(firstDay time.Time, closes ...string) []vo.PriceCandleVo {
	dailyCandles := []vo.PriceCandleVo{}
	for index, close := range closes {
		dailyCandles = append(dailyCandles, candleAt(firstDay.AddDate(0, 0, index), close))
	}
	return dailyCandles
}

// count daily candles closing at "100" followed by one closing at lastClose
func dailyCandlesEndingAt(firstDay time.Time, count int, lastClose string) []vo.PriceCandleVo {
	closes := []string{}
	for range count - 1 {
		closes = append(closes, "100")
	}
	return dailyCandlesFrom(firstDay, append(closes, lastClose)...)
}

func utcDay(day int) time.Time {
	return time.Date(2026, 10, day, 0, 0, 0, 0, time.UTC)
}

func backtestOf(grade string, timeHorizon string, completedAt time.Time) domains.AnalysisBacktestDomain {
	return domains.NewAnalysisBacktestDomain(entities.AnalysisResult{AnalysisEventID: 42, Symbol: "2330", Category: "twStock", Grade: grade, Confidence: 60, TimeHorizon: timeHorizon, CreatedAt: completedAt})
}

func decimalPointer(value string) *decimal.Decimal {
	parsed := decimal.RequireFromString(value)
	return &parsed
}

func TestAnalysisBacktestDomain_JudgesTheGradeAgainstTheCheckpointClose(t *testing.T) {
	// completed Wednesday 10:15 Taipei, so Thursday 2026-10-08 is the first trading day
	completedAt := taipeiAt(7, 10, 15)
	entryMinute := []vo.PriceCandleVo{candleAt(completedAt, "100")}
	testCases := []struct {
		name               string
		grade              string
		timeHorizon        string
		tradingDays        int
		checkpointClose    string
		expectedOutcome    string
		expectedChangeRate string
	}{
		{name: "bullish and rose", grade: "bullish", timeHorizon: "short", tradingDays: 10, checkpointClose: "105", expectedOutcome: "hit", expectedChangeRate: "0.05"},
		{name: "bearish and fell", grade: "bearish", timeHorizon: "mid", tradingDays: 60, checkpointClose: "92", expectedOutcome: "hit", expectedChangeRate: "-0.08"},
		{name: "strong bullish but fell", grade: "strongBullish", timeHorizon: "short", tradingDays: 10, checkpointClose: "97", expectedOutcome: "miss", expectedChangeRate: "-0.03"},
		{name: "bullish but unchanged", grade: "bullish", timeHorizon: "short", tradingDays: 10, checkpointClose: "100", expectedOutcome: "miss", expectedChangeRate: "0"},
		{name: "strong bearish but unchanged", grade: "strongBearish", timeHorizon: "short", tradingDays: 10, checkpointClose: "100", expectedOutcome: "miss", expectedChangeRate: "0"},
		{name: "strong bearish and fell", grade: "strongBearish", timeHorizon: "short", tradingDays: 10, checkpointClose: "99", expectedOutcome: "hit", expectedChangeRate: "-0.01"},
		{name: "bearish but rose", grade: "bearish", timeHorizon: "short", tradingDays: 10, checkpointClose: "101", expectedOutcome: "miss", expectedChangeRate: "0.01"},
		{name: "neutral is not judged", grade: "neutral", timeHorizon: "short", tradingDays: 10, checkpointClose: "103", expectedOutcome: "neutral", expectedChangeRate: "0.03"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			analysisBacktest := backtestOf(testCase.grade, testCase.timeHorizon, completedAt).Evaluate(vo.BacktestPricesVo{
				MinuteCandles: entryMinute,
				// the candle after the checkpoint must not be picked
				DailyCandles: append(dailyCandlesEndingAt(utcDay(8), testCase.tradingDays, testCase.checkpointClose), candleAt(utcDay(8).AddDate(0, 0, testCase.tradingDays), "500")),
			}, longAfterwards)

			assert.Equal(t, testCase.expectedOutcome, analysisBacktest.Outcome)
			require.NotNil(t, analysisBacktest.PriceChangeRate)
			assert.Equal(t, decimal.RequireFromString(testCase.expectedChangeRate).String(), analysisBacktest.PriceChangeRate.String())
			assert.Equal(t, decimal.RequireFromString(testCase.checkpointClose).String(), analysisBacktest.CheckpointPrice.String())
		})
	}
}

func TestAnalysisBacktestDomain_ReportsTheWholeAnalysis(t *testing.T) {
	completedAt := taipeiAt(7, 10, 15)

	analysisBacktest := backtestOf("bullish", "short", completedAt).Evaluate(vo.BacktestPricesVo{
		MinuteCandles: []vo.PriceCandleVo{candleAt(completedAt, "100")},
		DailyCandles:  dailyCandlesEndingAt(utcDay(8), 10, "105"),
	}, longAfterwards)

	checkpointDate := "2026-10-17"
	require.NotNil(t, analysisBacktest.PriceChangeRate)
	assert.True(t, decimal.RequireFromString("0.05").Equal(*analysisBacktest.PriceChangeRate))
	analysisBacktest.PriceChangeRate = nil
	assert.Equal(t, dto.AnalysisBacktestDto{
		AnalysisEventID: 42,
		Symbol:          "2330",
		Category:        "twStock",
		Grade:           "bullish",
		Confidence:      60,
		ConfidenceBand:  "medium",
		TimeHorizon:     "short",
		CompletedAt:     completedAt,
		EntryPrice:      decimalPointer("100"),
		CheckpointDate:  &checkpointDate,
		CheckpointPrice: decimalPointer("105"),
		Outcome:         "hit",
	}, analysisBacktest)
}

func TestAnalysisBacktestDomain_EntryPriceIsTheLastPriceByCompletion(t *testing.T) {
	// before any of these analyses could run out of days, so the entry price is reported
	withinTheWindow := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	testCases := []struct {
		name               string
		completedAt        time.Time
		prices             vo.BacktestPricesVo
		expectedEntryPrice string
	}{
		{
			name:        "during the session it is the completion minute",
			completedAt: taipeiAt(7, 10, 15),
			prices: vo.BacktestPricesVo{MinuteCandles: []vo.PriceCandleVo{
				candleAt(taipeiAt(7, 10, 14), "990"), candleAt(taipeiAt(7, 10, 15), "1000"), candleAt(taipeiAt(7, 10, 16), "1020"),
			}},
			expectedEntryPrice: "1000",
		},
		{
			name:               "after the close it is the last minute before completion",
			completedAt:        taipeiAt(7, 20, 30),
			prices:             vo.BacktestPricesVo{MinuteCandles: []vo.PriceCandleVo{candleAt(taipeiAt(7, 13, 28), "1005"), candleAt(taipeiAt(7, 13, 29), "1010")}},
			expectedEntryPrice: "1010",
		},
		{
			name:               "a weekend crypto analysis uses its own minute",
			completedAt:        time.Date(2026, 10, 10, 3, 0, 0, 0, backtestTaipei),
			prices:             vo.BacktestPricesVo{MinuteCandles: []vo.PriceCandleVo{candleAt(time.Date(2026, 10, 10, 3, 0, 0, 0, backtestTaipei), "86000")}},
			expectedEntryPrice: "86000",
		},
		{
			name:        "a minute candle wins over an hourly one",
			completedAt: taipeiAt(7, 10, 15),
			prices: vo.BacktestPricesVo{
				MinuteCandles: []vo.PriceCandleVo{candleAt(taipeiAt(7, 10, 15), "1000")},
				HourlyCandles: []vo.PriceCandleVo{candleAt(taipeiAt(7, 10, 0), "990")},
			},
			expectedEntryPrice: "1000",
		},
		{
			name:        "with no recent minute it falls back to the last hourly candle before completion",
			completedAt: taipeiAt(8, 7, 30),
			prices: vo.BacktestPricesVo{HourlyCandles: []vo.PriceCandleVo{
				candleAt(taipeiAt(7, 12, 0), "1000"), candleAt(taipeiAt(7, 13, 0), "1010"), candleAt(taipeiAt(8, 9, 0), "1030"),
			}},
			expectedEntryPrice: "1010",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			analysisBacktest := backtestOf("bullish", "short", testCase.completedAt).Evaluate(testCase.prices, withinTheWindow)

			require.NotNil(t, analysisBacktest.EntryPrice)
			assert.Equal(t, testCase.expectedEntryPrice, analysisBacktest.EntryPrice.String())
		})
	}
}

func TestAnalysisBacktestDomain_FirstTradingDayFollowsTheEntryPriceDay(t *testing.T) {
	testCases := []struct {
		name                   string
		completedAt            time.Time
		prices                 vo.BacktestPricesVo
		expectedCheckpointDate string
	}{
		{
			// Wednesday is not counted, so the tenth trading day is ten candles after it
			name:                   "completed during the session",
			completedAt:            taipeiAt(7, 10, 15),
			prices:                 vo.BacktestPricesVo{MinuteCandles: []vo.PriceCandleVo{candleAt(taipeiAt(7, 10, 15), "1000")}, DailyCandles: dailyCandlesEndingAt(utcDay(7), 11, "1100")},
			expectedCheckpointDate: "2026-10-17",
		},
		{
			// the entry price is Wednesday's close, so Thursday itself is the first trading day
			name:                   "completed before the Taiwan open",
			completedAt:            taipeiAt(8, 7, 30),
			prices:                 vo.BacktestPricesVo{HourlyCandles: []vo.PriceCandleVo{candleAt(taipeiAt(7, 13, 0), "1010")}, DailyCandles: dailyCandlesEndingAt(utcDay(7), 11, "1100")},
			expectedCheckpointDate: "2026-10-17",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			analysisBacktest := backtestOf("bullish", "short", testCase.completedAt).Evaluate(testCase.prices, longAfterwards)

			require.NotNil(t, analysisBacktest.CheckpointDate)
			assert.Equal(t, testCase.expectedCheckpointDate, *analysisBacktest.CheckpointDate)
		})
	}
}

func TestAnalysisBacktestDomain_PendingUntilTheHorizonHasPassed(t *testing.T) {
	completedAt := taipeiAt(7, 10, 15)

	analysisBacktest := backtestOf("bullish", "short", completedAt).Evaluate(vo.BacktestPricesVo{
		MinuteCandles: []vo.PriceCandleVo{candleAt(completedAt, "100")},
		DailyCandles:  dailyCandlesEndingAt(utcDay(7), 8, "105"),
	}, time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC))

	assert.Equal(t, "pending", analysisBacktest.Outcome)
	assert.Equal(t, decimalPointer("100"), analysisBacktest.EntryPrice)
	assert.Nil(t, analysisBacktest.CheckpointPrice)
	assert.Nil(t, analysisBacktest.CheckpointDate)
	assert.Nil(t, analysisBacktest.PriceChangeRate)
}

func TestAnalysisBacktestDomain_UnevaluableWithoutAUsableEntryPrice(t *testing.T) {
	completedAt := taipeiAt(7, 10, 15)
	testCases := []struct {
		name   string
		prices vo.BacktestPricesVo
	}{
		{name: "the price service has no candles for the symbol", prices: vo.BacktestPricesVo{}},
		{name: "every candle opened after completion", prices: vo.BacktestPricesVo{MinuteCandles: []vo.PriceCandleVo{candleAt(taipeiAt(7, 10, 16), "100")}, DailyCandles: dailyCandlesEndingAt(utcDay(8), 10, "105")}},
		{name: "the entry price is not positive", prices: vo.BacktestPricesVo{MinuteCandles: []vo.PriceCandleVo{candleAt(completedAt, "0")}, DailyCandles: dailyCandlesEndingAt(utcDay(8), 10, "105")}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			analysisBacktest := backtestOf("bullish", "short", completedAt).Evaluate(testCase.prices, longAfterwards)

			assert.Equal(t, "unevaluable", analysisBacktest.Outcome)
			assert.Nil(t, analysisBacktest.EntryPrice)
			assert.Nil(t, analysisBacktest.PriceChangeRate)
		})
	}
}

func TestAnalysisBacktestDomain_UnevaluableWhenTheMarketHasNoPriceSource(t *testing.T) {
	analysisBacktest := backtestOf("bullish", "short", taipeiAt(7, 10, 15)).Unevaluable()

	assert.Equal(t, "unevaluable", analysisBacktest.Outcome)
	assert.Equal(t, uint(42), analysisBacktest.AnalysisEventID)
	assert.Nil(t, analysisBacktest.EntryPrice)
}

func TestAnalysisBacktestDomain_IgnoresThePriceAtAnalysis(t *testing.T) {
	completedAt := taipeiAt(7, 10, 15)
	analysisResult := entities.AnalysisResult{Symbol: "2330", Category: "twStock", Grade: "bullish", TimeHorizon: "short", CreatedAt: completedAt, Price: decimal.NullDecimal{}}

	analysisBacktest := domains.NewAnalysisBacktestDomain(analysisResult).Evaluate(vo.BacktestPricesVo{
		MinuteCandles: []vo.PriceCandleVo{candleAt(completedAt, "100")},
		DailyCandles:  dailyCandlesEndingAt(utcDay(8), 10, "105"),
	}, longAfterwards)

	assert.Equal(t, "hit", analysisBacktest.Outcome)
}

func TestAnalysisBacktestDomain_ConfidenceBands(t *testing.T) {
	testCases := []struct {
		confidence   int
		expectedBand string
	}{
		{confidence: 0, expectedBand: "low"},
		{confidence: 49, expectedBand: "low"},
		{confidence: 50, expectedBand: "medium"},
		{confidence: 74, expectedBand: "medium"},
		{confidence: 75, expectedBand: "high"},
		{confidence: 100, expectedBand: "high"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.expectedBand, func(t *testing.T) {
			analysisResult := entities.AnalysisResult{Grade: "bullish", Confidence: testCase.confidence, TimeHorizon: "short"}

			analysisBacktest := domains.NewAnalysisBacktestDomain(analysisResult).Unevaluable()

			assert.Equal(t, testCase.expectedBand, analysisBacktest.ConfidenceBand)
		})
	}
}

func TestAnalysisBacktestDomain_PriceWindows(t *testing.T) {
	completedAt := taipeiAt(7, 10, 15)
	testCases := []struct {
		timeHorizon        string
		expectedDailyUntil time.Time
	}{
		// ten trading days span about fourteen calendar days, plus thirty for long holidays, to the end of that UTC day
		{timeHorizon: "short", expectedDailyUntil: time.Date(2026, 11, 20, 23, 59, 59, 0, time.UTC)},
		{timeHorizon: "mid", expectedDailyUntil: time.Date(2027, 1, 29, 23, 59, 59, 0, time.UTC)},
	}
	for _, testCase := range testCases {
		t.Run(testCase.timeHorizon, func(t *testing.T) {
			analysisResult := entities.AnalysisResult{Symbol: "BTC", Category: "crypto", TimeHorizon: testCase.timeHorizon, CreatedAt: completedAt}

			priceWindows := domains.NewAnalysisBacktestDomain(analysisResult).PriceWindows()

			assert.Equal(t, vo.BacktestPriceWindowsVo{
				Minute:         vo.PriceHistoryQueryVo{Symbol: "BTC", Category: "crypto", Interval: "1m", Since: completedAt.Add(-16 * time.Hour), Until: completedAt},
				HourlyFallback: vo.PriceHistoryQueryVo{Symbol: "BTC", Category: "crypto", Interval: "1h", Since: completedAt.AddDate(0, 0, -40), Until: completedAt},
				Daily:          vo.PriceHistoryQueryVo{Symbol: "BTC", Category: "crypto", Interval: "1d", Since: completedAt.AddDate(0, 0, -10), Until: testCase.expectedDailyUntil},
			}, priceWindows)
		})
	}
}

func TestAnalysisBacktestDomain_PriceChangeRateKeepsFourDecimalPlaces(t *testing.T) {
	completedAt := taipeiAt(7, 10, 15)

	analysisBacktest := backtestOf("bullish", "short", completedAt).Evaluate(vo.BacktestPricesVo{
		MinuteCandles: []vo.PriceCandleVo{candleAt(completedAt, "3")},
		DailyCandles:  dailyCandlesEndingAt(utcDay(8), 10, "4"),
	}, longAfterwards)

	require.NotNil(t, analysisBacktest.PriceChangeRate)
	assert.Equal(t, "0.3333", analysisBacktest.PriceChangeRate.String())
}

func TestAnalysisBacktestDomain_CheckpointDateIsTheDailyCandlesUtcDay(t *testing.T) {
	completedAt := taipeiAt(7, 10, 15)
	westOfUtc := time.FixedZone("UTC-5", -5*60*60)
	dailyCandles := dailyCandlesEndingAt(utcDay(8), 10, "105")
	dailyCandles[9].OpenTime = dailyCandles[9].OpenTime.In(westOfUtc)

	analysisBacktest := backtestOf("bullish", "short", completedAt).Evaluate(vo.BacktestPricesVo{
		MinuteCandles: []vo.PriceCandleVo{candleAt(completedAt, "100")},
		DailyCandles:  dailyCandles,
	}, longAfterwards)

	require.NotNil(t, analysisBacktest.CheckpointDate)
	assert.Equal(t, "2026-10-17", *analysisBacktest.CheckpointDate)
}

func TestAnalysisBacktestDomain_ACheckpointDayStillTradingIsNotYetDue(t *testing.T) {
	completedAt := taipeiAt(7, 10, 15)
	prices := vo.BacktestPricesVo{
		MinuteCandles: []vo.PriceCandleVo{candleAt(completedAt, "100")},
		// the tenth trading day is 2026-10-17
		DailyCandles: dailyCandlesEndingAt(utcDay(8), 10, "105"),
	}
	testCases := []struct {
		name            string
		category        string
		evaluatedAt     time.Time
		expectedOutcome string
	}{
		// a day counts an hour after its close, once its last minutes have reached the price service
		{name: "taiwan within an hour of its 13:30 close", category: "twStock", evaluatedAt: time.Date(2026, 10, 17, 14, 29, 0, 0, backtestTaipei), expectedOutcome: "pending"},
		{name: "taiwan an hour after its 13:30 close", category: "twStock", evaluatedAt: time.Date(2026, 10, 17, 14, 30, 0, 0, backtestTaipei), expectedOutcome: "hit"},
		{name: "crypto within an hour of the utc day ending", category: "crypto", evaluatedAt: time.Date(2026, 10, 18, 0, 59, 0, 0, time.UTC), expectedOutcome: "pending"},
		{name: "crypto an hour after the utc day ends", category: "crypto", evaluatedAt: time.Date(2026, 10, 18, 1, 0, 0, 0, time.UTC), expectedOutcome: "hit"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			analysisResult := entities.AnalysisResult{Symbol: "X", Category: testCase.category, Grade: "bullish", TimeHorizon: "short", CreatedAt: completedAt}

			analysisBacktest := domains.NewAnalysisBacktestDomain(analysisResult).Evaluate(prices, testCase.evaluatedAt)

			assert.Equal(t, testCase.expectedOutcome, analysisBacktest.Outcome)
		})
	}
}

func TestAnalysisBacktestDomain_DaysWithoutADailyPriceAreNotTradingDays(t *testing.T) {
	completedAt := taipeiAt(7, 10, 15)
	// Thursday 2026-10-08 through 2026-10-18, with 2026-10-10 a holiday that has no daily price
	dailyCandles := append(dailyCandlesFrom(utcDay(8), "100", "100"), dailyCandlesEndingAt(utcDay(11), 8, "105")...)

	analysisBacktest := backtestOf("bullish", "short", completedAt).Evaluate(vo.BacktestPricesVo{
		MinuteCandles: []vo.PriceCandleVo{candleAt(completedAt, "100")},
		DailyCandles:  dailyCandles,
	}, longAfterwards)

	require.NotNil(t, analysisBacktest.CheckpointDate)
	assert.Equal(t, "2026-10-18", *analysisBacktest.CheckpointDate)
}

func TestAnalysisBacktestDomain_UnevaluableOnceTheWholeDailyWindowHasPassed(t *testing.T) {
	completedAt := taipeiAt(7, 10, 15)
	// the price service stopped after seven trading days; the short window's last day is 2026-11-20, which
	// settles an hour after its 13:30 Taipei close
	prices := vo.BacktestPricesVo{MinuteCandles: []vo.PriceCandleVo{candleAt(completedAt, "100")}, DailyCandles: dailyCandlesEndingAt(utcDay(8), 7, "105")}
	testCases := []struct {
		name               string
		evaluatedAt        time.Time
		expectedOutcome    string
		expectedEntryPrice *decimal.Decimal
	}{
		{name: "the last day may still arrive", evaluatedAt: time.Date(2026, 11, 20, 14, 29, 59, 999000000, backtestTaipei), expectedOutcome: "pending", expectedEntryPrice: decimalPointer("100")},
		{name: "nothing more is coming", evaluatedAt: time.Date(2026, 11, 20, 14, 30, 0, 0, backtestTaipei), expectedOutcome: "unevaluable"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			analysisBacktest := backtestOf("bullish", "short", completedAt).Evaluate(prices, testCase.evaluatedAt)

			assert.Equal(t, testCase.expectedOutcome, analysisBacktest.Outcome)
			assert.Equal(t, testCase.expectedEntryPrice, analysisBacktest.EntryPrice)
		})
	}
}

func TestAnalysisBacktestDomain_OnlyAMarketThatClosesFallsBackToHourlyCandles(t *testing.T) {
	completedAt := taipeiAt(8, 7, 30)
	testCases := []struct {
		name             string
		category         string
		minuteCandles    []vo.PriceCandleVo
		expectedFallback bool
	}{
		{name: "taiwan with no recent minute", category: "twStock", expectedFallback: true},
		{name: "taiwan whose minutes all opened after completion", category: "twStock", minuteCandles: []vo.PriceCandleVo{candleAt(taipeiAt(8, 9, 0), "1030")}, expectedFallback: true},
		{name: "taiwan with a minute before completion", category: "twStock", minuteCandles: []vo.PriceCandleVo{candleAt(taipeiAt(8, 7, 29), "1010")}, expectedFallback: false},
		{name: "crypto with a gap in its minutes", category: "crypto", expectedFallback: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			analysisResult := entities.AnalysisResult{Symbol: "X", Category: testCase.category, Grade: "bullish", TimeHorizon: "short", CreatedAt: completedAt}

			needsHourlyFallback := domains.NewAnalysisBacktestDomain(analysisResult).NeedsHourlyFallback(testCase.minuteCandles)

			assert.Equal(t, testCase.expectedFallback, needsHourlyFallback)
		})
	}
}

func TestAnalysisBacktestDomain_CryptoWithAMinuteGapIsUnevaluableRatherThanStale(t *testing.T) {
	completedAt := taipeiAt(7, 10, 15)
	analysisResult := entities.AnalysisResult{Symbol: "BTC", Category: "crypto", Grade: "bullish", TimeHorizon: "short", CreatedAt: completedAt}

	analysisBacktest := domains.NewAnalysisBacktestDomain(analysisResult).Evaluate(vo.BacktestPricesVo{
		HourlyCandles: []vo.PriceCandleVo{candleAt(taipeiAt(5, 10, 0), "86000")},
		DailyCandles:  dailyCandlesEndingAt(utcDay(8), 10, "90000"),
	}, longAfterwards)

	assert.Equal(t, "unevaluable", analysisBacktest.Outcome)
	assert.Nil(t, analysisBacktest.EntryPrice)
}

func TestAnalysisBacktestDomain_AnUnevaluableResultCarriesNoPrices(t *testing.T) {
	completedAt := taipeiAt(7, 10, 15)

	analysisBacktest := backtestOf("bullish", "short", completedAt).Evaluate(vo.BacktestPricesVo{
		MinuteCandles: []vo.PriceCandleVo{candleAt(completedAt, "100")},
		DailyCandles:  dailyCandlesEndingAt(utcDay(8), 7, "105"),
	}, longAfterwards)

	assert.Equal(t, "unevaluable", analysisBacktest.Outcome)
	assert.Nil(t, analysisBacktest.EntryPrice)
	assert.Nil(t, analysisBacktest.CheckpointPrice)
}

func TestAnalysisBacktestDomain_ACryptoWindowEndsAnHourAfterItsLastUtcDay(t *testing.T) {
	completedAt := taipeiAt(7, 10, 15)
	analysisResult := entities.AnalysisResult{Symbol: "BTC", Category: "crypto", Grade: "bullish", TimeHorizon: "short", CreatedAt: completedAt}
	// the price service stopped after seven trading days; the short window's last day is 2026-11-20 UTC
	prices := vo.BacktestPricesVo{MinuteCandles: []vo.PriceCandleVo{candleAt(completedAt, "100")}, DailyCandles: dailyCandlesEndingAt(utcDay(8), 7, "105")}
	testCases := []struct {
		evaluatedAt     time.Time
		expectedOutcome string
	}{
		{evaluatedAt: time.Date(2026, 11, 21, 0, 59, 59, 0, time.UTC), expectedOutcome: "pending"},
		{evaluatedAt: time.Date(2026, 11, 21, 1, 0, 0, 0, time.UTC), expectedOutcome: "unevaluable"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.expectedOutcome, func(t *testing.T) {
			analysisBacktest := domains.NewAnalysisBacktestDomain(analysisResult).Evaluate(prices, testCase.evaluatedAt)

			assert.Equal(t, testCase.expectedOutcome, analysisBacktest.Outcome)
		})
	}
}

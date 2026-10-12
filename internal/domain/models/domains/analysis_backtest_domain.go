package domains

import (
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

const (
	BacktestOutcomeHit         = "hit"
	BacktestOutcomeMiss        = "miss"
	BacktestOutcomeNeutral     = "neutral"
	BacktestOutcomePending     = "pending"
	BacktestOutcomeUnevaluable = "unevaluable"

	ConfidenceBandLow    = "low"
	ConfidenceBandMedium = "medium"
	ConfidenceBandHigh   = "high"

	shortHorizonTradingDays     = 10
	midHorizonTradingDays       = 60
	mediumConfidenceFloor       = 50
	highConfidenceFloor         = 75
	priceChangeRatePlaces       = 4
	checkpointDateLayout        = "2006-01-02"
	dailyCandleLength           = 24 * time.Hour
	dailyLookbackDays           = 10
	dailyHolidayAllowanceDays   = 30
	entryMinuteLookback         = 16 * time.Hour
	entryHourlyFallbackLookback = 40 * 24 * time.Hour
	// the price service builds a day from its minute candles, so the last day can arrive a little late
	dailyCandleArrivalGrace = 24 * time.Hour
)

// the backtest of one completed user analysis: where its entry price comes from, which trading day
// is its checkpoint, and whether the price moved the way its grade said
type AnalysisBacktestDomain struct {
	analysisResult entities.AnalysisResult
}

func NewAnalysisBacktestDomain(analysisResult entities.AnalysisResult) AnalysisBacktestDomain {
	return AnalysisBacktestDomain{analysisResult: analysisResult}
}

// the minute window stays under the price service's 1000-candle limit; the hourly one is only asked for
// when the minute one has no entry price, see NeedsHourlyFallback
func (analysisBacktestDomain AnalysisBacktestDomain) PriceWindows() vo.BacktestPriceWindowsVo {
	analysisResult := analysisBacktestDomain.analysisResult
	completedAt := analysisResult.CreatedAt
	priceHistoryQuery := func(interval string, since time.Time, until time.Time) vo.PriceHistoryQueryVo {
		return vo.PriceHistoryQueryVo{Symbol: analysisResult.Symbol, Category: analysisResult.Category, Interval: interval, Since: since, Until: until}
	}
	return vo.BacktestPriceWindowsVo{
		Minute:         priceHistoryQuery(vo.PriceCandleIntervalMinute, completedAt.Add(-entryMinuteLookback), completedAt),
		HourlyFallback: priceHistoryQuery(vo.PriceCandleIntervalHour, completedAt.Add(-entryHourlyFallbackLookback), completedAt),
		Daily:          priceHistoryQuery(vo.PriceCandleIntervalDay, completedAt.AddDate(0, 0, -dailyLookbackDays), analysisBacktestDomain.dailyWindowEnd()),
	}
}

// only a market that closes falls back to hourly candles: when nothing traded in the minute window
// (a weekend, a long holiday) the last hourly candle is the last price there was, whereas a gap in a
// market trading around the clock is missing data and would hand over a stale price
func (analysisBacktestDomain AnalysisBacktestDomain) NeedsHourlyFallback(minuteCandles []vo.PriceCandleVo) bool {
	_, hasEntry := analysisBacktestDomain.entryCandleIn(minuteCandles)
	return !hasEntry && analysisBacktestDomain.analysisResult.Category == vo.MarketCategoryTwStock
}

// evaluatedAt decides which daily candles are finished: the price service also returns the day still trading
func (analysisBacktestDomain AnalysisBacktestDomain) Evaluate(backtestPrices vo.BacktestPricesVo, evaluatedAt time.Time) dto.AnalysisBacktestDto {
	entryCandle, hasEntry := analysisBacktestDomain.entryCandleIn(backtestPrices.MinuteCandles)
	if analysisBacktestDomain.NeedsHourlyFallback(backtestPrices.MinuteCandles) {
		entryCandle, hasEntry = analysisBacktestDomain.entryCandleIn(backtestPrices.HourlyCandles)
	}
	if !hasEntry || !entryCandle.Close.IsPositive() {
		return analysisBacktestDomain.toDto(BacktestOutcomeUnevaluable, nil, nil)
	}
	// the first trading day is the one after the day the entry price belongs to, so an analysis finished
	// before the Taiwan open counts its own day; daily candles are cut at UTC midnight
	entryDay := entryCandle.OpenTime.Truncate(dailyCandleLength)
	// Taiwan's daily candle, cut at UTC midnight, carries the Taipei date of its session
	dayClosesAt := func(candleDay time.Time) time.Time {
		if analysisBacktestDomain.analysisResult.Category != vo.MarketCategoryTwStock {
			// crypto trades around the clock, so its day closes only when the UTC day ends
			return candleDay.Add(dailyCandleLength)
		}
		taipeiDate := candleDay.In(taipeiTimeZone)
		return time.Date(taipeiDate.Year(), taipeiDate.Month(), taipeiDate.Day(), 0, 0, 0, 0, taipeiTimeZone).Add(taiwanStockMarketClose)
	}
	tradingDaysAfterEntry := 0
	for _, dailyCandle := range backtestPrices.DailyCandles {
		candleDay := dailyCandle.OpenTime.Truncate(dailyCandleLength)
		if !candleDay.After(entryDay) {
			continue
		}
		// a day still trading has no close yet, so it cannot be a checkpoint
		if dayClosesAt(candleDay).After(evaluatedAt) {
			break
		}
		tradingDaysAfterEntry++
		if tradingDaysAfterEntry == analysisBacktestDomain.requiredTradingDays() {
			outcome := BacktestOutcomeNeutral
			switch analysisBacktestDomain.analysisResult.Grade {
			case GradeStrongBullish, GradeBullish:
				outcome = BacktestOutcomeMiss
				if dailyCandle.Close.GreaterThan(entryCandle.Close) {
					outcome = BacktestOutcomeHit
				}
			case GradeStrongBearish, GradeBearish:
				outcome = BacktestOutcomeMiss
				if dailyCandle.Close.LessThan(entryCandle.Close) {
					outcome = BacktestOutcomeHit
				}
			}
			return analysisBacktestDomain.toDto(outcome, &entryCandle, &dailyCandle)
		}
	}
	// once the whole daily window, and the time its last day needs to arrive, are behind us, the missing days
	// are not coming: the symbol stopped trading or the price service stopped collecting it
	if !evaluatedAt.Before(analysisBacktestDomain.dailyWindowEnd().Add(time.Second + dailyCandleArrivalGrace)) {
		return analysisBacktestDomain.toDto(BacktestOutcomeUnevaluable, nil, nil)
	}
	return analysisBacktestDomain.toDto(BacktestOutcomePending, &entryCandle, nil)
}

// used when the market has no price source to ask
func (analysisBacktestDomain AnalysisBacktestDomain) Unevaluable() dto.AnalysisBacktestDto {
	return analysisBacktestDomain.toDto(BacktestOutcomeUnevaluable, nil, nil)
}

func (analysisBacktestDomain AnalysisBacktestDomain) requiredTradingDays() int {
	if analysisBacktestDomain.analysisResult.TimeHorizon == TimeHorizonMid {
		return midHorizonTradingDays
	}
	return shortHorizonTradingDays
}

// roughly five trading days a week plus room for long holidays, running to the end of that UTC day so the
// last daily candle is a whole day rather than one cut off at the completion time of day
func (analysisBacktestDomain AnalysisBacktestDomain) dailyWindowEnd() time.Time {
	lastDay := analysisBacktestDomain.analysisResult.CreatedAt.AddDate(0, 0, analysisBacktestDomain.requiredTradingDays()*7/5+dailyHolidayAllowanceDays)
	return lastDay.UTC().Truncate(dailyCandleLength).Add(dailyCandleLength - time.Second)
}

// the last candle opened by completion, latest first
func (analysisBacktestDomain AnalysisBacktestDomain) entryCandleIn(candles []vo.PriceCandleVo) (vo.PriceCandleVo, bool) {
	for index := len(candles) - 1; index >= 0; index-- {
		if !candles[index].OpenTime.After(analysisBacktestDomain.analysisResult.CreatedAt) {
			return candles[index], true
		}
	}
	return vo.PriceCandleVo{}, false
}

func (analysisBacktestDomain AnalysisBacktestDomain) toDto(outcome string, entryCandle *vo.PriceCandleVo, checkpointCandle *vo.PriceCandleVo) dto.AnalysisBacktestDto {
	analysisResult := analysisBacktestDomain.analysisResult
	confidenceBand := ConfidenceBandHigh
	if analysisResult.Confidence < mediumConfidenceFloor {
		confidenceBand = ConfidenceBandLow
	} else if analysisResult.Confidence < highConfidenceFloor {
		confidenceBand = ConfidenceBandMedium
	}
	analysisBacktest := dto.AnalysisBacktestDto{
		AnalysisEventID: analysisResult.AnalysisEventID,
		Symbol:          analysisResult.Symbol,
		Category:        analysisResult.Category,
		Grade:           analysisResult.Grade,
		Confidence:      analysisResult.Confidence,
		ConfidenceBand:  confidenceBand,
		TimeHorizon:     analysisResult.TimeHorizon,
		CompletedAt:     analysisResult.CreatedAt,
		Outcome:         outcome,
	}
	if entryCandle != nil {
		analysisBacktest.EntryPrice = &entryCandle.Close
	}
	if entryCandle != nil && checkpointCandle != nil {
		checkpointDate := checkpointCandle.OpenTime.UTC().Format(checkpointDateLayout)
		priceChangeRate := checkpointCandle.Close.Sub(entryCandle.Close).Div(entryCandle.Close).Round(priceChangeRatePlaces)
		analysisBacktest.CheckpointDate = &checkpointDate
		analysisBacktest.CheckpointPrice = &checkpointCandle.Close
		analysisBacktest.PriceChangeRate = &priceChangeRate
	}
	return analysisBacktest
}

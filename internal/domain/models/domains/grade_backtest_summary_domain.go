package domains

import (
	"math"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/shopspring/decimal"
)

const (
	RandomGuessHitRate = 0.5
	hitRatePrecision   = 10000
)

var (
	directionalGrades = []string{GradeStrongBullish, GradeBullish, GradeBearish, GradeStrongBearish}
	backtestOutcomes  = []string{BacktestOutcomeHit, BacktestOutcomeMiss, BacktestOutcomeNeutral, BacktestOutcomePending, BacktestOutcomeUnevaluable}
	marketCategories  = []string{vo.MarketCategoryTwStock, vo.MarketCategoryUsStock, vo.MarketCategoryCrypto}
	confidenceBands   = []string{ConfidenceBandLow, ConfidenceBandMedium, ConfidenceBandHigh}
)

// every grouping always lists all of its groups, so an empty group reads as "no data" instead of going missing
type GradeBacktestSummaryDomain struct {
	analysisBacktests []dto.AnalysisBacktestDto
}

func NewGradeBacktestSummaryDomain(analysisBacktests []dto.AnalysisBacktestDto) GradeBacktestSummaryDomain {
	return GradeBacktestSummaryDomain{analysisBacktests: analysisBacktests}
}

func (gradeBacktestSummaryDomain GradeBacktestSummaryDomain) ToDto() dto.GradeBacktestDto {
	tallies := map[string]*dto.HitRateDto{"": {}}
	for _, groups := range [][]string{marketCategories, directionalGrades, confidenceBands} {
		for _, group := range groups {
			tallies[group] = &dto.HitRateDto{}
		}
	}
	outcomeCounts := map[string]int{}
	for _, backtestOutcome := range backtestOutcomes {
		outcomeCounts[backtestOutcome] = 0
	}
	neutralChangeRateSum := decimal.Zero
	neutralCount := 0
	for _, analysisBacktest := range gradeBacktestSummaryDomain.analysisBacktests {
		outcomeCounts[analysisBacktest.Outcome]++
		if analysisBacktest.Outcome == BacktestOutcomeNeutral {
			neutralChangeRateSum = neutralChangeRateSum.Add(*analysisBacktest.PriceChangeRate)
			neutralCount++
		}
		if analysisBacktest.Outcome != BacktestOutcomeHit && analysisBacktest.Outcome != BacktestOutcomeMiss {
			continue
		}
		// the overall tally sits under the empty key so it is counted the same way as every group
		for _, group := range []string{"", analysisBacktest.Category, analysisBacktest.Grade, analysisBacktest.ConfidenceBand} {
			if analysisBacktest.Outcome == BacktestOutcomeHit {
				tallies[group].HitCount++
			} else {
				tallies[group].MissCount++
			}
		}
	}
	for _, tally := range tallies {
		if judgedCount := tally.HitCount + tally.MissCount; judgedCount > 0 {
			hitRate := math.Round(float64(tally.HitCount)/float64(judgedCount)*hitRatePrecision) / hitRatePrecision
			tally.HitRate = &hitRate
		}
	}
	groupTallies := func(groups []string) map[string]dto.HitRateDto {
		groupedTallies := map[string]dto.HitRateDto{}
		for _, group := range groups {
			groupedTallies[group] = *tallies[group]
		}
		return groupedTallies
	}
	gradeBacktest := dto.GradeBacktestDto{
		RandomBaseline:   RandomGuessHitRate,
		Overall:          *tallies[""],
		ByCategory:       groupTallies(marketCategories),
		ByGrade:          groupTallies(directionalGrades),
		ByConfidenceBand: groupTallies(confidenceBands),
		OutcomeCounts:    outcomeCounts,
		Analyses:         append([]dto.AnalysisBacktestDto{}, gradeBacktestSummaryDomain.analysisBacktests...),
	}
	if neutralCount > 0 {
		neutralAverageChangeRate := neutralChangeRateSum.Div(decimal.NewFromInt(int64(neutralCount))).Round(priceChangeRatePlaces)
		gradeBacktest.NeutralAverageChangeRate = &neutralAverageChangeRate
	}
	return gradeBacktest
}

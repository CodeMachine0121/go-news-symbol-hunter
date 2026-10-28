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

	groupingOverall        = "overall"
	groupingCategory       = "category"
	groupingGrade          = "grade"
	groupingConfidenceBand = "confidenceBand"
	groupingSource         = "source"
)

var (
	directionalGrades = []string{GradeStrongBullish, GradeBullish, GradeBearish, GradeStrongBearish}
	backtestOutcomes  = []string{BacktestOutcomeHit, BacktestOutcomeMiss, BacktestOutcomeNeutral, BacktestOutcomePending, BacktestOutcomeUnevaluable}
	marketCategories  = []string{vo.MarketCategoryTwStock, vo.MarketCategoryUsStock, vo.MarketCategoryCrypto}
	confidenceBands   = []string{ConfidenceBandLow, ConfidenceBandMedium, ConfidenceBandHigh}
	gradeSources      = []string{vo.GradeSourceManual, vo.GradeSourceBackground}
)

// every grouping always lists all of its groups, so an empty group reads as "no data" instead of going missing
type GradeBacktestSummaryDomain struct {
	analysisBacktests []dto.AnalysisBacktestDto
}

func NewGradeBacktestSummaryDomain(analysisBacktests []dto.AnalysisBacktestDto) GradeBacktestSummaryDomain {
	return GradeBacktestSummaryDomain{analysisBacktests: analysisBacktests}
}

func (gradeBacktestSummaryDomain GradeBacktestSummaryDomain) ToDto() dto.GradeBacktestDto {
	// each grouping keeps its own tallies, so a value shared by two groupings can never be counted twice
	groupings := map[string][]string{
		groupingCategory:       marketCategories,
		groupingGrade:          directionalGrades,
		groupingConfidenceBand: confidenceBands,
		groupingSource:         gradeSources,
	}
	tallies := map[string]map[string]*dto.HitRateDto{groupingOverall: {"": {}}}
	for grouping, groups := range groupings {
		tallies[grouping] = map[string]*dto.HitRateDto{}
		for _, group := range groups {
			tallies[grouping][group] = &dto.HitRateDto{}
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
		for _, groupOf := range [...][2]string{
			{groupingOverall, ""},
			{groupingCategory, analysisBacktest.Category},
			{groupingGrade, analysisBacktest.Grade},
			{groupingConfidenceBand, analysisBacktest.ConfidenceBand},
			{groupingSource, analysisBacktest.Source},
		} {
			tally, isListed := tallies[groupOf[0]][groupOf[1]]
			if !isListed {
				continue
			}
			if analysisBacktest.Outcome == BacktestOutcomeHit {
				tally.HitCount++
			} else {
				tally.MissCount++
			}
		}
	}
	groupTallies := func(grouping string) map[string]dto.HitRateDto {
		groupedTallies := map[string]dto.HitRateDto{}
		for group, tally := range tallies[grouping] {
			if judgedCount := tally.HitCount + tally.MissCount; judgedCount > 0 {
				hitRate := math.Round(float64(tally.HitCount)/float64(judgedCount)*hitRatePrecision) / hitRatePrecision
				tally.HitRate = &hitRate
			}
			groupedTallies[group] = *tally
		}
		return groupedTallies
	}
	gradeBacktest := dto.GradeBacktestDto{
		RandomBaseline:   RandomGuessHitRate,
		Overall:          groupTallies(groupingOverall)[""],
		ByCategory:       groupTallies(groupingCategory),
		ByGrade:          groupTallies(groupingGrade),
		ByConfidenceBand: groupTallies(groupingConfidenceBand),
		BySource:         groupTallies(groupingSource),
		OutcomeCounts:    outcomeCounts,
		Analyses:         append([]dto.AnalysisBacktestDto{}, gradeBacktestSummaryDomain.analysisBacktests...),
	}
	if neutralCount > 0 {
		neutralAverageChangeRate := neutralChangeRateSum.Div(decimal.NewFromInt(int64(neutralCount))).Round(priceChangeRatePlaces)
		gradeBacktest.NeutralAverageChangeRate = &neutralAverageChangeRate
	}
	return gradeBacktest
}

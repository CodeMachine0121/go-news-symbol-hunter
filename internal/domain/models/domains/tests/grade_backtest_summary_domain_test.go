package domains_test

import (
	"testing"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func judgedBacktests(count int, outcome string, category string, grade string, confidenceBand string) []dto.AnalysisBacktestDto {
	analysisBacktests := []dto.AnalysisBacktestDto{}
	for range count {
		analysisBacktests = append(analysisBacktests, dto.AnalysisBacktestDto{Outcome: outcome, Category: category, Grade: grade, ConfidenceBand: confidenceBand, PriceChangeRate: decimalPointer("0.01")})
	}
	return analysisBacktests
}

func hitRateOf(hitCount int, missCount int, hitRate float64) dto.HitRateDto {
	return dto.HitRateDto{HitCount: hitCount, MissCount: missCount, HitRate: &hitRate}
}

func TestGradeBacktestSummaryDomain_OverallHitRateBesideRandomGuessing(t *testing.T) {
	analysisBacktests := append(judgedBacktests(26, "hit", "twStock", "bullish", "high"), judgedBacktests(14, "miss", "twStock", "bullish", "high")...)

	gradeBacktest := domains.NewGradeBacktestSummaryDomain(analysisBacktests).ToDto()

	assert.Equal(t, hitRateOf(26, 14, 0.65), gradeBacktest.Overall)
	assert.Equal(t, 0.5, gradeBacktest.RandomBaseline)
}

func TestGradeBacktestSummaryDomain_GroupsByCategoryGradeAndConfidence(t *testing.T) {
	analysisBacktests := append(append(append(
		judgedBacktests(3, "hit", "twStock", "bullish", "high"),
		judgedBacktests(1, "miss", "twStock", "strongBearish", "medium")...),
		judgedBacktests(1, "hit", "crypto", "bearish", "low")...),
		judgedBacktests(2, "miss", "crypto", "bullish", "high")...)

	gradeBacktest := domains.NewGradeBacktestSummaryDomain(analysisBacktests).ToDto()

	assert.Equal(t, map[string]dto.HitRateDto{
		"twStock": hitRateOf(3, 1, 0.75),
		"crypto":  hitRateOf(1, 2, 0.3333),
		"usStock": {},
	}, gradeBacktest.ByCategory)
	assert.Equal(t, map[string]dto.HitRateDto{
		"strongBullish": {},
		"bullish":       hitRateOf(3, 2, 0.6),
		"bearish":       hitRateOf(1, 0, 1),
		"strongBearish": hitRateOf(0, 1, 0),
	}, gradeBacktest.ByGrade)
	assert.Equal(t, map[string]dto.HitRateDto{
		"low":    hitRateOf(1, 0, 1),
		"medium": hitRateOf(0, 1, 0),
		"high":   hitRateOf(3, 2, 0.6),
	}, gradeBacktest.ByConfidenceBand)
}

func TestGradeBacktestSummaryDomain_AGroupWithNothingJudgedHasNoData(t *testing.T) {
	analysisBacktests := append(judgedBacktests(1, "hit", "twStock", "bullish", "high"), judgedBacktests(2, "pending", "twStock", "bullish", "low")...)

	gradeBacktest := domains.NewGradeBacktestSummaryDomain(analysisBacktests).ToDto()

	assert.Nil(t, gradeBacktest.ByConfidenceBand["low"].HitRate)
	assert.Equal(t, 0, gradeBacktest.ByConfidenceBand["low"].HitCount+gradeBacktest.ByConfidenceBand["low"].MissCount)
}

func TestGradeBacktestSummaryDomain_NeutralGradesOnlyContributeTheirAverageChange(t *testing.T) {
	analysisBacktests := []dto.AnalysisBacktestDto{
		{Outcome: "neutral", Category: "twStock", Grade: "neutral", ConfidenceBand: "low", PriceChangeRate: decimalPointer("0.03")},
		{Outcome: "neutral", Category: "twStock", Grade: "neutral", ConfidenceBand: "low", PriceChangeRate: decimalPointer("0.0125")},
		{Outcome: "pending", Category: "twStock", Grade: "neutral", ConfidenceBand: "low"},
	}

	gradeBacktest := domains.NewGradeBacktestSummaryDomain(analysisBacktests).ToDto()

	require.NotNil(t, gradeBacktest.NeutralAverageChangeRate)
	assert.Equal(t, "0.0213", gradeBacktest.NeutralAverageChangeRate.String())
	assert.Equal(t, dto.HitRateDto{}, gradeBacktest.Overall)
}

func TestGradeBacktestSummaryDomain_CountsEveryOutcomeAndListsEveryAnalysis(t *testing.T) {
	analysisBacktests := append(append(append(
		judgedBacktests(2, "hit", "twStock", "bullish", "high"),
		judgedBacktests(1, "pending", "twStock", "bullish", "high")...),
		judgedBacktests(3, "unevaluable", "usStock", "bearish", "low")...),
		judgedBacktests(1, "miss", "crypto", "bearish", "low")...)

	gradeBacktest := domains.NewGradeBacktestSummaryDomain(analysisBacktests).ToDto()

	assert.Equal(t, map[string]int{"hit": 2, "miss": 1, "neutral": 0, "pending": 1, "unevaluable": 3}, gradeBacktest.OutcomeCounts)
	assert.Equal(t, analysisBacktests, gradeBacktest.Analyses)
	assert.Nil(t, gradeBacktest.NeutralAverageChangeRate)
}

func TestGradeBacktestSummaryDomain_NothingToBacktest(t *testing.T) {
	gradeBacktest := domains.NewGradeBacktestSummaryDomain(nil).ToDto()

	assert.Equal(t, dto.HitRateDto{}, gradeBacktest.Overall)
	assert.Equal(t, []dto.AnalysisBacktestDto{}, gradeBacktest.Analyses)
	assert.Len(t, gradeBacktest.ByConfidenceBand, 3)
}

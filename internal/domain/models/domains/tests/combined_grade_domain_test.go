package domains_test

import (
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/stretchr/testify/assert"
)

var combinedAt = time.Date(2026, 10, 7, 13, 0, 0, 0, time.UTC)

func TestCombinedGradeDomain_WeightsTheSessionsTheSymbolAlreadyHas(t *testing.T) {
	defaultWeights := vo.NewSessionWeightsVo(0.3, 0.2, 0.5)
	testCases := []struct {
		name               string
		sessionGrades      []entities.SessionGrade
		expectedScore      float64
		expectedGrade      string
		expectedConfidence int
	}{
		{name: "only pre-market equals pre-market", sessionGrades: []entities.SessionGrade{{Session: "preMarket", Grade: "bullish", Confidence: 60}}, expectedScore: 1, expectedGrade: "bullish", expectedConfidence: 60},
		{name: "pre-market and intraday", sessionGrades: []entities.SessionGrade{{Session: "preMarket", Grade: "bullish", Confidence: 60}, {Session: "intraday", Grade: "bearish", Confidence: 40}}, expectedScore: 0.2, expectedGrade: "neutral", expectedConfidence: 52},
		{name: "all three sessions", sessionGrades: []entities.SessionGrade{{Session: "preMarket", Grade: "strongBullish"}, {Session: "intraday", Grade: "bullish"}, {Session: "afterMarket", Grade: "bearish"}}, expectedScore: 0.3, expectedGrade: "neutral"},
		{name: "a missing session is left out", sessionGrades: []entities.SessionGrade{{Session: "preMarket", Grade: "bullish"}, {Session: "afterMarket", Grade: "bullish"}}, expectedScore: 1, expectedGrade: "bullish"},
		{name: "combined confidence rounds to the nearest whole number", sessionGrades: []entities.SessionGrade{{Session: "preMarket", Grade: "neutral", Confidence: 61}, {Session: "intraday", Grade: "neutral", Confidence: 40}}, expectedScore: 0, expectedGrade: "neutral", expectedConfidence: 53},
		{name: "no session grades is neutral", sessionGrades: nil, expectedScore: 0, expectedGrade: "neutral"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			gradeRecord := domains.NewCombinedGradeDomain(testCase.sessionGrades, defaultWeights).ToGradeRecordEntity(vo.SymbolVo{Value: "2330", Category: vo.MarketCategoryVo{Value: "twStock"}}, vo.NewSymbolNameVo("台積電"), "2026-10-07", combinedAt)

			assert.Equal(t, testCase.expectedGrade, gradeRecord.Grade)
			assert.Equal(t, &testCase.expectedScore, gradeRecord.CombinedScore)
			assert.Equal(t, testCase.expectedConfidence, gradeRecord.Confidence)
		})
	}
}

func TestCombinedGradeDomain_ScoreThresholds(t *testing.T) {
	equalWeights := vo.NewSessionWeightsVo(1, 1, 1)
	testCases := []struct {
		score         float64
		preMarket     string
		afterMarket   string
		weights       vo.SessionWeightsVo
		expectedGrade string
	}{
		{score: 1.5, preMarket: "strongBullish", afterMarket: "bullish", weights: equalWeights, expectedGrade: "strongBullish"},
		{score: 0.5, preMarket: "bullish", afterMarket: "neutral", weights: equalWeights, expectedGrade: "bullish"},
		{score: 0.4, preMarket: "bullish", afterMarket: "neutral", weights: vo.NewSessionWeightsVo(0.4, 1, 0.6), expectedGrade: "neutral"},
		{score: -0.4, preMarket: "bearish", afterMarket: "neutral", weights: vo.NewSessionWeightsVo(0.4, 1, 0.6), expectedGrade: "neutral"},
		{score: -0.5, preMarket: "bearish", afterMarket: "neutral", weights: equalWeights, expectedGrade: "bearish"},
		{score: -1.5, preMarket: "strongBearish", afterMarket: "bearish", weights: equalWeights, expectedGrade: "strongBearish"},
	}
	for _, testCase := range testCases {
		gradeRecord := domains.NewCombinedGradeDomain([]entities.SessionGrade{{Session: "preMarket", Grade: testCase.preMarket}, {Session: "afterMarket", Grade: testCase.afterMarket}}, testCase.weights).ToGradeRecordEntity(vo.SymbolVo{}, vo.SymbolNameVo{}, "", combinedAt)

		assert.Equal(t, testCase.score, *gradeRecord.CombinedScore)
		assert.Equal(t, testCase.expectedGrade, gradeRecord.Grade, "score %v", testCase.score)
	}
}

func TestCombinedGradeDomain_BecomesABackgroundGradeRecord(t *testing.T) {
	testCases := []struct {
		name               string
		symbolName         vo.SymbolNameVo
		expectedSymbolName string
	}{
		{name: "carries the registered name", symbolName: vo.NewSymbolNameVo("台積電"), expectedSymbolName: "台積電"},
		{name: "no registered name", symbolName: vo.NewSymbolNameVo(""), expectedSymbolName: ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			sessionGrades := []entities.SessionGrade{{Session: "preMarket", Grade: "bullish", Confidence: 60}}

			gradeRecord := domains.NewCombinedGradeDomain(sessionGrades, vo.NewSessionWeightsVo(0.3, 0.2, 0.5)).ToGradeRecordEntity(vo.SymbolVo{Value: "2330", Category: vo.MarketCategoryVo{Value: "twStock"}}, testCase.symbolName, "2026-10-07", combinedAt)

			combinedScore := 1.0
			assert.Equal(t, entities.GradeRecord{
				Source: "background", Symbol: "2330", Category: "twStock", SymbolName: testCase.expectedSymbolName, TradingDay: "2026-10-07",
				Grade: "bullish", Confidence: 60, TimeHorizon: "short", CombinedScore: &combinedScore,
				KeyEvents: []entities.AnalysisKeyEvent{}, RiskFactors: []string{}, Evidence: []entities.AnalysisEvidence{}, CreatedAt: combinedAt,
			}, gradeRecord)
			assert.Nil(t, gradeRecord.AnalysisEventID)
			assert.Empty(t, gradeRecord.Reason)
			assert.False(t, gradeRecord.Price.Valid)
		})
	}
}

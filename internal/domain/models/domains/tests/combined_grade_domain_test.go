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
			combinedGrade := domains.NewCombinedGradeDomain(testCase.sessionGrades, defaultWeights).ToEntity(vo.SymbolVo{Value: "2330", Category: vo.MarketCategoryVo{Value: "twStock"}}, "2026-10-07", combinedAt)

			assert.Equal(t, entities.CombinedGrade{Symbol: "2330", Category: "twStock", TradingDay: "2026-10-07", Grade: testCase.expectedGrade, CombinedScore: testCase.expectedScore, Confidence: testCase.expectedConfidence, UpdatedAt: combinedAt}, combinedGrade)
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
		combinedGrade := domains.NewCombinedGradeDomain([]entities.SessionGrade{{Session: "preMarket", Grade: testCase.preMarket}, {Session: "afterMarket", Grade: testCase.afterMarket}}, testCase.weights).ToEntity(vo.SymbolVo{}, "", combinedAt)

		assert.Equal(t, testCase.score, combinedGrade.CombinedScore)
		assert.Equal(t, testCase.expectedGrade, combinedGrade.Grade, "score %v", testCase.score)
	}
}

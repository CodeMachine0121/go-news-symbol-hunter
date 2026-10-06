package domains_test

import (
	"strings"
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

var analyzedAt = time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

func finishedBefore(duration time.Duration) *time.Time {
	finishedAt := analyzedAt.Add(-duration)
	return &finishedAt
}

func TestAnalysisEventDomain_IsReusableAt(t *testing.T) {
	testCases := []struct {
		name          string
		analysisEvent entities.AnalysisEvent
		expected      bool
	}{
		{name: "running for 15 minutes", analysisEvent: entities.AnalysisEvent{Status: "running", StartedAt: analyzedAt.Add(-15 * time.Minute)}, expected: true},
		{name: "running for 15 minutes and 1 second", analysisEvent: entities.AnalysisEvent{Status: "running", StartedAt: analyzedAt.Add(-15*time.Minute - time.Second)}, expected: false},
		{name: "succeeded 5h59m ago", analysisEvent: entities.AnalysisEvent{Status: "succeeded", FinishedAt: finishedBefore(5*time.Hour + 59*time.Minute)}, expected: true},
		{name: "succeeded exactly 6h ago", analysisEvent: entities.AnalysisEvent{Status: "succeeded", FinishedAt: finishedBefore(6 * time.Hour)}, expected: true},
		{name: "succeeded 6h1m ago", analysisEvent: entities.AnalysisEvent{Status: "succeeded", FinishedAt: finishedBefore(6*time.Hour + time.Minute)}, expected: false},
		{name: "succeeded without finish time", analysisEvent: entities.AnalysisEvent{Status: "succeeded"}, expected: false},
		{name: "failed 1h ago", analysisEvent: entities.AnalysisEvent{Status: "failed", FinishedAt: finishedBefore(time.Hour)}, expected: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, domains.NewAnalysisEventDomain(testCase.analysisEvent).IsReusableAt(analyzedAt))
		})
	}
}

func TestAnalysisEventDomain_StartsRunningAndFinishesWithUsage(t *testing.T) {
	crypto, _ := vo.NewMarketCategoryVo("crypto")
	symbol, _ := vo.NewSymbolVo("btc", crypto)
	analysisEvent := domains.NewStartedAnalysisEventDomain(7, symbol, "claude-opus-5-5", analyzedAt)
	assert.Equal(t, entities.AnalysisEvent{ApiKeyID: 7, Symbol: "BTC", Category: "crypto", Status: "running", Model: "claude-opus-5-5", StartedAt: analyzedAt}, analysisEvent.ToEntity())

	analysisEvent.Fail("AI 服務暫時無法使用", analyzedAt.Add(time.Minute), vo.AnalystUsageVo{InputTokens: 100, OutputTokens: 20})

	finishedAt := analyzedAt.Add(time.Minute)
	assert.Equal(t, entities.AnalysisEvent{ApiKeyID: 7, Symbol: "BTC", Category: "crypto", Status: "failed", FailureReason: "AI 服務暫時無法使用", Model: "claude-opus-5-5", InputTokens: 100, OutputTokens: 20, StartedAt: analyzedAt, FinishedAt: &finishedAt}, analysisEvent.ToEntity())
	analysisEvent.Succeed(finishedAt, vo.AnalystUsageVo{InputTokens: 5, OutputTokens: 6})
	assert.Equal(t, "succeeded", analysisEvent.ToEntity().Status)
	assert.Empty(t, analysisEvent.ToEntity().FailureReason)
	assert.True(t, analysisEvent.IsSucceeded())
}

func TestAnalysisEventDomain_ToDto(t *testing.T) {
	finishedAt := analyzedAt.Add(time.Minute)
	analysisEvent := domains.NewAnalysisEventDomain(entities.AnalysisEvent{ID: 3, Symbol: "BTC", Category: "crypto", Status: "succeeded", StartedAt: analyzedAt, FinishedAt: &finishedAt})
	analysisResult := entities.AnalysisResult{
		AnalysisEventID: 3, Symbol: "BTC", Category: "crypto", Grade: "bullish", Confidence: 70, TimeHorizon: "short", Reason: "理由",
		KeyEvents:   []entities.AnalysisKeyEvent{{Title: "t", Link: "l", PublishedAt: analyzedAt}},
		RiskFactors: []string{"風險"},
		Evidence:    []entities.AnalysisEvidence{{Title: "t", Link: "l", PublishedAt: analyzedAt, ProviderName: "CoinDesk"}},
		CreatedAt:   finishedAt,
	}

	assert.Equal(t, dto.AnalysisEventDto{AnalysisEventID: 3, Symbol: "BTC", Category: "crypto", Status: "succeeded", StartedAt: analyzedAt, FinishedAt: &finishedAt}, analysisEvent.ToDto(nil))
	assert.Equal(t, &dto.AnalysisResultDto{
		AnalysisEventID: 3, Symbol: "BTC", Category: "crypto", Grade: "bullish", Confidence: 70, TimeHorizon: "short", Reason: "理由",
		KeyEvents:   []dto.AnalysisKeyEventDto{{Title: "t", Link: "l", PublishedAt: analyzedAt}},
		RiskFactors: []string{"風險"},
		Evidence:    []dto.AnalysisEvidenceDto{{Title: "t", Link: "l", PublishedAt: analyzedAt, ProviderName: "CoinDesk"}},
		CreatedAt:   finishedAt,
	}, analysisEvent.ToDto(&analysisResult).Result)
}

func recordedEvidence(links ...string) *domains.AnalysisEvidenceDomain {
	analysisEvidence := domains.NewAnalysisEvidenceDomain()
	news := []dto.NewsDto{}
	for _, link := range links {
		news = append(news, dto.NewsDto{Title: "title " + link, Link: link, PublishedAt: analyzedAt, ProviderName: "CoinDesk"})
	}
	analysisEvidence.Record(news)
	return analysisEvidence
}

func conclude(t *testing.T, rawConclusion vo.RawAnalysisConclusionVo, analysisEvidence *domains.AnalysisEvidenceDomain) entities.AnalysisResult {
	conclusion, err := domains.NewAnalysisConclusionDomain(rawConclusion, analysisEvidence)
	require.NoError(t, err)
	return conclusion.ToResultEntity(entities.AnalysisEvent{ID: 9, Symbol: "BTC", Category: "crypto"}, analyzedAt, nil)
}

func TestAnalysisConclusionDomain_NormalizesTheConclusion(t *testing.T) {
	testCases := []struct {
		name                string
		rawConclusion       vo.RawAnalysisConclusionVo
		expectedGrade       string
		expectedConfidence  int
		expectedTimeHorizon string
	}{
		{name: "valid values are kept", rawConclusion: vo.RawAnalysisConclusionVo{Grade: "strongBearish", Confidence: 42, TimeHorizon: "mid", Reason: "r"}, expectedGrade: "strongBearish", expectedConfidence: 42, expectedTimeHorizon: "mid"},
		{name: "unknown grade becomes neutral", rawConclusion: vo.RawAnalysisConclusionVo{Grade: "超級看多", Confidence: 50, TimeHorizon: "short", Reason: "r"}, expectedGrade: "neutral", expectedConfidence: 50, expectedTimeHorizon: "short"},
		{name: "confidence above 100 becomes 100", rawConclusion: vo.RawAnalysisConclusionVo{Grade: "bullish", Confidence: 130, TimeHorizon: "short", Reason: "r"}, expectedGrade: "bullish", expectedConfidence: 100, expectedTimeHorizon: "short"},
		{name: "confidence below 0 becomes 0", rawConclusion: vo.RawAnalysisConclusionVo{Grade: "bullish", Confidence: -5, TimeHorizon: "short", Reason: "r"}, expectedGrade: "bullish", expectedConfidence: 0, expectedTimeHorizon: "short"},
		{name: "unknown time horizon becomes short", rawConclusion: vo.RawAnalysisConclusionVo{Grade: "bullish", Confidence: 50, TimeHorizon: "長期", Reason: "r"}, expectedGrade: "bullish", expectedConfidence: 50, expectedTimeHorizon: "short"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			analysisResult := conclude(t, testCase.rawConclusion, recordedEvidence("a"))

			assert.Equal(t, testCase.expectedGrade, analysisResult.Grade)
			assert.Equal(t, testCase.expectedConfidence, analysisResult.Confidence)
			assert.Equal(t, testCase.expectedTimeHorizon, analysisResult.TimeHorizon)
		})
	}
}

func TestAnalysisConclusionDomain_WithoutEvidenceIsNeutralWithZeroConfidence(t *testing.T) {
	analysisResult := conclude(t, vo.RawAnalysisConclusionVo{Grade: "bullish", Confidence: 80, TimeHorizon: "short", Reason: "r"}, domains.NewAnalysisEvidenceDomain())

	assert.Equal(t, "neutral", analysisResult.Grade)
	assert.Equal(t, 0, analysisResult.Confidence)
	assert.Equal(t, []entities.AnalysisEvidence{}, analysisResult.Evidence)
}

func TestAnalysisConclusionDomain_KeyEventsCiteOnlyEvidenceUpToFive(t *testing.T) {
	analysisResult := conclude(t, vo.RawAnalysisConclusionVo{
		Grade: "bullish", Confidence: 60, TimeHorizon: "short", Reason: "r",
		KeyEventLinks: []string{"a", "made-up", "a", " b ", "c", "d", "e", "f"},
	}, recordedEvidence("a", "b", "c", "d", "e", "f"))

	links := []string{}
	for _, keyEvent := range analysisResult.KeyEvents {
		links = append(links, keyEvent.Link)
	}
	assert.Equal(t, []string{"a", "b", "c", "d", "e"}, links)
	assert.Equal(t, entities.AnalysisKeyEvent{Title: "title a", Link: "a", PublishedAt: analyzedAt}, analysisResult.KeyEvents[0])
}

func TestAnalysisConclusionDomain_RiskFactorsSkipBlanksUpToFive(t *testing.T) {
	analysisResult := conclude(t, vo.RawAnalysisConclusionVo{
		Grade: "bullish", Confidence: 60, TimeHorizon: "short", Reason: "r",
		RiskFactors: []string{"一", "  ", "二", "三", "四", "五", "六"},
	}, recordedEvidence("a"))

	assert.Equal(t, []string{"一", "二", "三", "四", "五"}, analysisResult.RiskFactors)
}

func TestAnalysisConclusionDomain_RequiresAReason(t *testing.T) {
	_, err := domains.NewAnalysisConclusionDomain(vo.RawAnalysisConclusionVo{Grade: "bullish", Reason: "   "}, recordedEvidence("a"))

	assert.ErrorIs(t, err, domains.ErrAnalysisIncomplete)
	assert.EqualError(t, err, "AI 未提供完整分析")
}

func TestAnalysisConclusionDomain_ToResultEntityCarriesTheEventAndEvidence(t *testing.T) {
	analysisResult := conclude(t, vo.RawAnalysisConclusionVo{Grade: "bearish", Confidence: 30, TimeHorizon: "mid", Reason: " 理由 "}, recordedEvidence("a", "a", "b"))

	assert.Equal(t, uint(9), analysisResult.AnalysisEventID)
	assert.Equal(t, "BTC", analysisResult.Symbol)
	assert.Equal(t, "crypto", analysisResult.Category)
	assert.Equal(t, "理由", analysisResult.Reason)
	assert.Equal(t, analyzedAt, analysisResult.CreatedAt)
	assert.Equal(t, []entities.AnalysisEvidence{
		{Title: "title a", Link: "a", PublishedAt: analyzedAt, ProviderName: "CoinDesk"},
		{Title: "title b", Link: "b", PublishedAt: analyzedAt, ProviderName: "CoinDesk"},
	}, analysisResult.Evidence)
	assert.Equal(t, []entities.AnalysisKeyEvent{}, analysisResult.KeyEvents)
	assert.Equal(t, []string{}, analysisResult.RiskFactors)
	assert.False(t, strings.Contains(analysisResult.Reason, " "))
}

func TestAnalysisEventDomain_RecordsTheAnsweringModel(t *testing.T) {
	analysisEvent := domains.NewAnalysisEventDomain(entities.AnalysisEvent{Model: "claude-opus-5-5"})

	analysisEvent.RecordAnsweringModel("")
	unchangedModel := analysisEvent.ToEntity().Model
	analysisEvent.RecordAnsweringModel("claude-opus-4-8")

	assert.Equal(t, "claude-opus-5-5", unchangedModel)
	assert.Equal(t, "claude-opus-4-8", analysisEvent.ToEntity().Model)
}

func TestAnalysisEventDomain_IsStaleAt(t *testing.T) {
	assert.False(t, domains.NewAnalysisEventDomain(entities.AnalysisEvent{Status: "running", StartedAt: analyzedAt.Add(-15 * time.Minute)}).IsStaleAt(analyzedAt))
	assert.True(t, domains.NewAnalysisEventDomain(entities.AnalysisEvent{Status: "running", StartedAt: analyzedAt.Add(-15*time.Minute - time.Second)}).IsStaleAt(analyzedAt))
	assert.False(t, domains.NewAnalysisEventDomain(entities.AnalysisEvent{Status: "failed", StartedAt: analyzedAt.Add(-time.Hour)}).IsStaleAt(analyzedAt))
}

func TestAnalysisConclusionDomain_ToResultEntityCarriesThePriceQuote(t *testing.T) {
	conclusion, err := domains.NewAnalysisConclusionDomain(vo.RawAnalysisConclusionVo{Grade: "bullish", Reason: "r"}, recordedEvidence("a"))
	require.NoError(t, err)
	priceQuote, err := vo.NewPriceQuoteVo(decimal.RequireFromString("0.00001234"), "USDT", analyzedAt, "Binance")
	require.NoError(t, err)

	analysisResult := conclusion.ToResultEntity(entities.AnalysisEvent{ID: 9}, analyzedAt, &priceQuote)
	analysisResultWithoutPrice := conclusion.ToResultEntity(entities.AnalysisEvent{ID: 9}, analyzedAt, nil)

	assert.Equal(t, "0.00001234", analysisResult.Price.Decimal.String())
	assert.True(t, analysisResult.Price.Valid)
	assert.Equal(t, "USDT", analysisResult.PriceCurrency)
	assert.Equal(t, &analyzedAt, analysisResult.PricedAt)
	assert.Equal(t, "Binance", analysisResult.PriceSource)
	assert.False(t, analysisResultWithoutPrice.Price.Valid)
	assert.Nil(t, analysisResultWithoutPrice.PricedAt)
}

func TestAnalysisEventDomain_ToDtoShowsThePriceAtAnalysisOrNull(t *testing.T) {
	analysisEvent := domains.NewAnalysisEventDomain(entities.AnalysisEvent{ID: 3, Status: "succeeded"})
	pricedResult := entities.AnalysisResult{AnalysisEventID: 3, Price: decimal.NewNullDecimal(decimal.RequireFromString("86607.62")), PriceCurrency: "USDT", PricedAt: &analyzedAt, PriceSource: "Binance"}
	unpricedResult := entities.AnalysisResult{AnalysisEventID: 3}

	assert.Equal(t, &dto.PriceAtAnalysisDto{Price: decimal.RequireFromString("86607.62"), Currency: "USDT", PricedAt: analyzedAt, Source: "Binance"}, analysisEvent.ToDto(&pricedResult).Result.PriceAtAnalysis)
	assert.Nil(t, analysisEvent.ToDto(&unpricedResult).Result.PriceAtAnalysis)
}

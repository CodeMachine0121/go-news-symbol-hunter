package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/service"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/persistence"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var analysisStartedAt = time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

func TestAnalysisEventRepository_AllowsOnlyOneRunningAnalysisPerSymbol(t *testing.T) {
	analysisEventRepository := persistence.NewAnalysisEventRepository(openTestDatabase(t))
	require.NoError(t, analysisEventRepository.Create(context.Background(), &entities.AnalysisEvent{Symbol: "BTC", Category: "crypto", Status: "running", StartedAt: analysisStartedAt}))

	duplicateError := analysisEventRepository.Create(context.Background(), &entities.AnalysisEvent{Symbol: "BTC", Category: "crypto", Status: "running", StartedAt: analysisStartedAt})
	otherMarketError := analysisEventRepository.Create(context.Background(), &entities.AnalysisEvent{Symbol: "BTC", Category: "usStock", Status: "running", StartedAt: analysisStartedAt})
	finishedError := analysisEventRepository.Create(context.Background(), &entities.AnalysisEvent{Symbol: "BTC", Category: "crypto", Status: "failed", StartedAt: analysisStartedAt})

	assert.ErrorIs(t, duplicateError, service.ErrAnalysisAlreadyRunning)
	assert.NoError(t, otherMarketError)
	assert.NoError(t, finishedError)
}

func TestAnalysisEventRepository_FindLatestReusableIgnoresFailedAndOtherSymbols(t *testing.T) {
	analysisEventRepository := persistence.NewAnalysisEventRepository(openTestDatabase(t))
	ctx := context.Background()
	olderSucceeded := entities.AnalysisEvent{Symbol: "BTC", Category: "crypto", Status: "succeeded", StartedAt: analysisStartedAt.Add(-3 * time.Hour)}
	newerSucceeded := entities.AnalysisEvent{Symbol: "BTC", Category: "crypto", Status: "succeeded", StartedAt: analysisStartedAt.Add(-2 * time.Hour)}
	require.NoError(t, analysisEventRepository.Create(ctx, &olderSucceeded))
	require.NoError(t, analysisEventRepository.Create(ctx, &newerSucceeded))
	require.NoError(t, analysisEventRepository.Create(ctx, &entities.AnalysisEvent{Symbol: "BTC", Category: "crypto", Status: "failed", StartedAt: analysisStartedAt.Add(-time.Hour)}))
	require.NoError(t, analysisEventRepository.Create(ctx, &entities.AnalysisEvent{Symbol: "ETH", Category: "crypto", Status: "running", StartedAt: analysisStartedAt}))

	latest, err := analysisEventRepository.FindLatestReusable(ctx, "BTC", "crypto")
	missing, missingError := analysisEventRepository.FindLatestReusable(ctx, "BTC", "usStock")

	require.NoError(t, err)
	require.NotNil(t, latest)
	assert.Equal(t, newerSucceeded.ID, latest.ID)
	assert.NoError(t, missingError)
	assert.Nil(t, missing)
}

func TestAnalysisEventRepository_UpdateAndFindByID(t *testing.T) {
	analysisEventRepository := persistence.NewAnalysisEventRepository(openTestDatabase(t))
	ctx := context.Background()
	analysisEvent := entities.AnalysisEvent{ApiKeyID: 3, Symbol: "BTC", Category: "crypto", Status: "running", Model: "claude-opus-5-5", StartedAt: analysisStartedAt}
	require.NoError(t, analysisEventRepository.Create(ctx, &analysisEvent))
	finishedAt := analysisStartedAt.Add(time.Minute)
	analysisEvent.Status = "succeeded"
	analysisEvent.FinishedAt = &finishedAt
	analysisEvent.InputTokens = 300
	analysisEvent.OutputTokens = 30

	require.NoError(t, analysisEventRepository.Update(ctx, &analysisEvent))
	foundAnalysisEvent, err := analysisEventRepository.FindByID(ctx, analysisEvent.ID)
	missingAnalysisEvent, missingError := analysisEventRepository.FindByID(ctx, analysisEvent.ID+100)

	require.NoError(t, err)
	assert.Equal(t, "succeeded", foundAnalysisEvent.Status)
	assert.Equal(t, int64(300), foundAnalysisEvent.InputTokens)
	assert.True(t, foundAnalysisEvent.FinishedAt.Equal(finishedAt))
	assert.NoError(t, missingError)
	assert.Nil(t, missingAnalysisEvent)
}

func TestAnalysisEventRepository_FailAllRunningLeavesFinishedEventsAlone(t *testing.T) {
	analysisEventRepository := persistence.NewAnalysisEventRepository(openTestDatabase(t))
	ctx := context.Background()
	runningEvent := entities.AnalysisEvent{Symbol: "BTC", Category: "crypto", Status: "running", StartedAt: analysisStartedAt}
	succeededEvent := entities.AnalysisEvent{Symbol: "ETH", Category: "crypto", Status: "succeeded", StartedAt: analysisStartedAt}
	require.NoError(t, analysisEventRepository.Create(ctx, &runningEvent))
	require.NoError(t, analysisEventRepository.Create(ctx, &succeededEvent))

	require.NoError(t, analysisEventRepository.FailAllRunning(ctx, "服務重新啟動，分析中斷", analysisStartedAt.Add(time.Hour)))
	failedEvent, _ := analysisEventRepository.FindByID(ctx, runningEvent.ID)
	untouchedEvent, _ := analysisEventRepository.FindByID(ctx, succeededEvent.ID)

	assert.Equal(t, "failed", failedEvent.Status)
	assert.Equal(t, "服務重新啟動，分析中斷", failedEvent.FailureReason)
	assert.True(t, failedEvent.FinishedAt.Equal(analysisStartedAt.Add(time.Hour)))
	assert.Equal(t, "succeeded", untouchedEvent.Status)
}

func TestAnalysisResultRepository_StoresStructuredFields(t *testing.T) {
	analysisResultRepository := persistence.NewAnalysisResultRepository(openTestDatabase(t))
	ctx := context.Background()
	analysisResult := entities.AnalysisResult{
		AnalysisEventID: 7, Symbol: "BTC", Category: "crypto", Grade: "bullish", Confidence: 70, TimeHorizon: "short", Reason: "理由",
		KeyEvents:   []entities.AnalysisKeyEvent{{Title: "t", Link: "https://news/1", PublishedAt: analysisStartedAt}},
		RiskFactors: []string{"風險"},
		Evidence:    []entities.AnalysisEvidence{{Title: "t", Link: "https://news/1", PublishedAt: analysisStartedAt, ProviderName: "CoinDesk"}},
	}

	require.NoError(t, analysisResultRepository.Create(ctx, &analysisResult))
	foundAnalysisResult, err := analysisResultRepository.FindByAnalysisEventID(ctx, 7)
	missingAnalysisResult, missingError := analysisResultRepository.FindByAnalysisEventID(ctx, 8)
	duplicateError := analysisResultRepository.Create(ctx, &entities.AnalysisResult{AnalysisEventID: 7, Symbol: "BTC", Category: "crypto", Grade: "neutral", TimeHorizon: "short", Reason: "r", KeyEvents: []entities.AnalysisKeyEvent{}, RiskFactors: []string{}, Evidence: []entities.AnalysisEvidence{}})

	require.NoError(t, err)
	assert.Equal(t, "bullish", foundAnalysisResult.Grade)
	assert.Equal(t, []string{"風險"}, foundAnalysisResult.RiskFactors)
	assert.Equal(t, "https://news/1", foundAnalysisResult.KeyEvents[0].Link)
	assert.True(t, foundAnalysisResult.KeyEvents[0].PublishedAt.Equal(analysisStartedAt))
	assert.Equal(t, "CoinDesk", foundAnalysisResult.Evidence[0].ProviderName)
	assert.NoError(t, missingError)
	assert.Nil(t, missingAnalysisResult)
	assert.Error(t, duplicateError)
}

func TestAnalysisRepositories_ReportStorageFailures(t *testing.T) {
	database := openTestDatabase(t)
	sqlDatabase, err := database.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDatabase.Close())
	ctx := context.Background()
	analysisEventRepository := persistence.NewAnalysisEventRepository(database)
	analysisResultRepository := persistence.NewAnalysisResultRepository(database)

	_, findError := analysisEventRepository.FindByID(ctx, 1)
	_, latestError := analysisEventRepository.FindLatestReusable(ctx, "BTC", "crypto")
	_, resultError := analysisResultRepository.FindByAnalysisEventID(ctx, 1)
	createError := analysisEventRepository.Create(ctx, &entities.AnalysisEvent{Symbol: "BTC", Category: "crypto", Status: "running", StartedAt: analysisStartedAt})

	assert.Error(t, findError)
	assert.Error(t, latestError)
	assert.Error(t, resultError)
	assert.Error(t, createError)
	assert.NotErrorIs(t, createError, service.ErrAnalysisAlreadyRunning)
}

func TestAnalysisResultRepository_StoresThePriceAtAnalysisExactly(t *testing.T) {
	analysisResultRepository := persistence.NewAnalysisResultRepository(openTestDatabase(t))
	ctx := context.Background()
	pricedResult := entities.AnalysisResult{
		AnalysisEventID: 1, Symbol: "SHIB", Category: "crypto", Grade: "neutral", TimeHorizon: "short", Reason: "r",
		KeyEvents: []entities.AnalysisKeyEvent{}, RiskFactors: []string{}, Evidence: []entities.AnalysisEvidence{},
		Price: decimal.NewNullDecimal(decimal.RequireFromString("0.000012345678901234")), PriceCurrency: "USDT", PricedAt: &analysisStartedAt, PriceSource: "Binance",
	}
	unpricedResult := entities.AnalysisResult{
		AnalysisEventID: 2, Symbol: "XYZ", Category: "crypto", Grade: "neutral", TimeHorizon: "short", Reason: "r",
		KeyEvents: []entities.AnalysisKeyEvent{}, RiskFactors: []string{}, Evidence: []entities.AnalysisEvidence{},
	}
	require.NoError(t, analysisResultRepository.Create(ctx, &pricedResult))
	require.NoError(t, analysisResultRepository.Create(ctx, &unpricedResult))

	foundPricedResult, err := analysisResultRepository.FindByAnalysisEventID(ctx, 1)
	foundUnpricedResult, unpricedError := analysisResultRepository.FindByAnalysisEventID(ctx, 2)

	require.NoError(t, err)
	require.NoError(t, unpricedError)
	assert.Equal(t, "0.000012345678901234", foundPricedResult.Price.Decimal.String())
	assert.True(t, foundPricedResult.PricedAt.Equal(analysisStartedAt))
	assert.Equal(t, "USDT", foundPricedResult.PriceCurrency)
	assert.Equal(t, "Binance", foundPricedResult.PriceSource)
	assert.False(t, foundUnpricedResult.Price.Valid)
	assert.Nil(t, foundUnpricedResult.PricedAt)
}

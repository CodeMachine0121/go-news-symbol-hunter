package persistence_test

import (
	"context"
	"testing"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/persistence"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositories_StoreTheSymbolName(t *testing.T) {
	database := openTestDatabase(t)
	ctx := context.Background()
	analysisEvent := entities.AnalysisEvent{Symbol: "2330", Category: "twStock", SymbolName: "台積電", Status: "succeeded", StartedAt: analysisStartedAt}
	require.NoError(t, persistence.NewAnalysisEventRepository(database).Create(ctx, &analysisEvent))
	require.NoError(t, persistence.NewAnalysisResultRepository(database).Create(ctx, &entities.AnalysisResult{AnalysisEventID: analysisEvent.ID, Symbol: "2330", Category: "twStock", SymbolName: "台積電", Grade: "bullish", Reason: "理由", KeyEvents: []entities.AnalysisKeyEvent{}, RiskFactors: []string{}, Evidence: []entities.AnalysisEvidence{}}))
	require.NoError(t, database.Create(&entities.TrackedSymbol{Symbol: "2330", Category: "twStock", SymbolName: "台積電", IsTracking: true}).Error)

	storedAnalysisEvent, err := persistence.NewAnalysisEventRepository(database).FindByID(ctx, analysisEvent.ID)
	require.NoError(t, err)
	storedAnalysisResult, err := persistence.NewAnalysisResultRepository(database).FindByAnalysisEventID(ctx, analysisEvent.ID)
	require.NoError(t, err)
	trackingSymbols, err := persistence.NewTrackedSymbolRepository(database).FindAllTracking(ctx)
	require.NoError(t, err)

	assert.Equal(t, "台積電", storedAnalysisEvent.SymbolName)
	assert.Equal(t, "台積電", storedAnalysisResult.SymbolName)
	require.Len(t, trackingSymbols, 1)
	assert.Equal(t, "台積電", trackingSymbols[0].SymbolName)
}

func TestRepositories_RowsFromBeforeSymbolNamesReadAnEmptyName(t *testing.T) {
	database := openTestDatabase(t)
	ctx := context.Background()
	require.NoError(t, database.Migrator().DropColumn(&entities.AnalysisEvent{}, "SymbolName"))
	require.NoError(t, database.Migrator().DropColumn(&entities.TrackedSymbol{}, "SymbolName"))
	legacyAnalysisEvent := entities.AnalysisEvent{Symbol: "2330", Category: "twStock", Status: "failed", StartedAt: analysisStartedAt}
	require.NoError(t, database.Omit("SymbolName").Create(&legacyAnalysisEvent).Error)
	require.NoError(t, database.Omit("SymbolName").Create(&entities.TrackedSymbol{Symbol: "2330", Category: "twStock", IsTracking: true}).Error)

	require.NoError(t, database.AutoMigrate(&entities.AnalysisEvent{}, &entities.TrackedSymbol{}))
	storedAnalysisEvent, err := persistence.NewAnalysisEventRepository(database).FindByID(ctx, legacyAnalysisEvent.ID)
	require.NoError(t, err)
	trackingSymbols, err := persistence.NewTrackedSymbolRepository(database).FindAllTracking(ctx)
	require.NoError(t, err)

	assert.Equal(t, "", storedAnalysisEvent.SymbolName)
	require.Len(t, trackingSymbols, 1)
	assert.Equal(t, "", trackingSymbols[0].SymbolName)
}

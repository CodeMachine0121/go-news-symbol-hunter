package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/service"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/persistence"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

var gradedAt = time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC)

func TestTrackedSymbolRepository_FindsOnlyTrackingSymbolsOfTheCategory(t *testing.T) {
	database := openTestDatabase(t)
	trackedSymbolRepository := persistence.NewTrackedSymbolRepository(database)
	require.NoError(t, database.Create(&entities.TrackedSymbol{Symbol: "2330", Category: "twStock", IsTracking: true}).Error)
	require.NoError(t, database.Create(&entities.TrackedSymbol{Symbol: "AAPL", Category: "usStock", IsTracking: true}).Error)
	paused := entities.TrackedSymbol{Symbol: "2317", Category: "twStock", IsTracking: true}
	require.NoError(t, database.Create(&paused).Error)
	require.NoError(t, database.Model(&paused).Update("is_tracking", false).Error)

	trackedSymbols, err := trackedSymbolRepository.FindTracking(context.Background(), "twStock")

	require.NoError(t, err)
	require.Len(t, trackedSymbols, 1)
	assert.Equal(t, "2330", trackedSymbols[0].Symbol)
}

func TestTrackedSymbolRepository_RegistersASymbolOnlyOncePerCategory(t *testing.T) {
	database := openTestDatabase(t)
	require.NoError(t, database.Create(&entities.TrackedSymbol{Symbol: "2330", Category: "twStock", IsTracking: true}).Error)

	duplicateError := database.Create(&entities.TrackedSymbol{Symbol: "2330", Category: "twStock", IsTracking: true}).Error
	otherCategoryError := database.Create(&entities.TrackedSymbol{Symbol: "2330", Category: "usStock", IsTracking: true}).Error

	assert.Error(t, duplicateError)
	assert.NoError(t, otherCategoryError)
}

func TestSessionGradeRepository_SaveReplacesTheSameSessionAndKeepsOthers(t *testing.T) {
	sessionGradeRepository := persistence.NewSessionGradeRepository(openTestDatabase(t))
	ctx := context.Background()
	require.NoError(t, sessionGradeRepository.Save(ctx, &entities.SessionGrade{Symbol: "2330", Category: "twStock", TradingDay: "2026-10-07", Session: "preMarket", Grade: "bullish", Confidence: 60, Reason: "first", KeyEvents: []entities.AnalysisKeyEvent{}, RiskFactors: []string{}, Evidence: []entities.AnalysisEvidence{}, CreatedAt: gradedAt}))
	require.NoError(t, sessionGradeRepository.Save(ctx, &entities.SessionGrade{Symbol: "2330", Category: "twStock", TradingDay: "2026-10-07", Session: "preMarket", Grade: "bearish", Confidence: 30, Reason: "rerun", KeyEvents: []entities.AnalysisKeyEvent{}, RiskFactors: []string{"匯率"}, Evidence: []entities.AnalysisEvidence{}, CreatedAt: gradedAt.Add(time.Minute)}))
	require.NoError(t, sessionGradeRepository.Save(ctx, &entities.SessionGrade{Symbol: "2330", Category: "twStock", TradingDay: "2026-10-07", Session: "intraday", Grade: "neutral", Reason: "intraday", KeyEvents: []entities.AnalysisKeyEvent{}, RiskFactors: []string{}, Evidence: []entities.AnalysisEvidence{}, CreatedAt: gradedAt.Add(time.Hour)}))
	require.NoError(t, sessionGradeRepository.Save(ctx, &entities.SessionGrade{Symbol: "2317", Category: "twStock", TradingDay: "2026-10-07", Session: "preMarket", Grade: "neutral", Reason: "other symbol", KeyEvents: []entities.AnalysisKeyEvent{}, RiskFactors: []string{}, Evidence: []entities.AnalysisEvidence{}, CreatedAt: gradedAt}))

	sessionGrades, err := sessionGradeRepository.FindByTradingDay(ctx, "2330", "twStock", "2026-10-07")

	require.NoError(t, err)
	require.Len(t, sessionGrades, 2)
	assert.Equal(t, "preMarket", sessionGrades[0].Session)
	assert.Equal(t, "bearish", sessionGrades[0].Grade)
	assert.Equal(t, []string{"匯率"}, sessionGrades[0].RiskFactors)
	assert.Equal(t, "intraday", sessionGrades[1].Session)
}

func TestSessionGradeRepository_DeleteExceptTradingDayKeepsOnlyThatDay(t *testing.T) {
	sessionGradeRepository := persistence.NewSessionGradeRepository(openTestDatabase(t))
	ctx := context.Background()
	for _, tradingDay := range []string{"2026-10-06", "2026-10-07"} {
		require.NoError(t, sessionGradeRepository.Save(ctx, &entities.SessionGrade{Symbol: "2330", Category: "twStock", TradingDay: tradingDay, Session: "afterMarket", Grade: "neutral", Reason: "r", KeyEvents: []entities.AnalysisKeyEvent{}, RiskFactors: []string{}, Evidence: []entities.AnalysisEvidence{}, CreatedAt: gradedAt}))
	}

	require.NoError(t, sessionGradeRepository.DeleteExceptTradingDay(ctx, "2026-10-07"))

	yesterday, err := sessionGradeRepository.FindByTradingDay(ctx, "2330", "twStock", "2026-10-06")
	require.NoError(t, err)
	today, err := sessionGradeRepository.FindByTradingDay(ctx, "2330", "twStock", "2026-10-07")
	require.NoError(t, err)
	assert.Empty(t, yesterday)
	assert.Len(t, today, 1)
}

func TestCombinedGradeRepository_SaveReplacesTheDayAndFindAllFilters(t *testing.T) {
	combinedGradeRepository := persistence.NewCombinedGradeRepository(openTestDatabase(t))
	ctx := context.Background()
	require.NoError(t, combinedGradeRepository.Save(ctx, &entities.CombinedGrade{Symbol: "2330", Category: "twStock", TradingDay: "2026-10-07", Grade: "bullish", CombinedScore: 1, Confidence: 60, UpdatedAt: gradedAt}))
	require.NoError(t, combinedGradeRepository.Save(ctx, &entities.CombinedGrade{Symbol: "2330", Category: "twStock", TradingDay: "2026-10-07", Grade: "neutral", CombinedScore: 0.2, Confidence: 52, UpdatedAt: gradedAt.Add(time.Hour)}))
	require.NoError(t, combinedGradeRepository.Save(ctx, &entities.CombinedGrade{Symbol: "2330", Category: "twStock", TradingDay: "2026-10-06", Grade: "bearish", CombinedScore: -1, UpdatedAt: gradedAt}))
	require.NoError(t, combinedGradeRepository.Save(ctx, &entities.CombinedGrade{Symbol: "2317", Category: "twStock", TradingDay: "2026-10-07", Grade: "neutral", UpdatedAt: gradedAt}))

	all, err := combinedGradeRepository.FindAll(ctx, "", "")
	require.NoError(t, err)
	only2330, err := combinedGradeRepository.FindAll(ctx, "2330", "twStock")
	require.NoError(t, err)
	otherCategory, err := combinedGradeRepository.FindAll(ctx, "2330", "usStock")
	require.NoError(t, err)

	require.Len(t, all, 3)
	assert.Equal(t, []string{"2317 2026-10-07", "2330 2026-10-07", "2330 2026-10-06"}, []string{all[0].Symbol + " " + all[0].TradingDay, all[1].Symbol + " " + all[1].TradingDay, all[2].Symbol + " " + all[2].TradingDay})
	assert.Equal(t, "neutral", all[1].Grade)
	assert.Equal(t, 0.2, all[1].CombinedScore)
	assert.Equal(t, 52, all[1].Confidence)
	assert.Len(t, only2330, 2)
	assert.Empty(t, otherCategory)
}

func TestCombinedGradeRepository_DeleteExceptTradingDayKeepsOnlyThatDay(t *testing.T) {
	combinedGradeRepository := persistence.NewCombinedGradeRepository(openTestDatabase(t))
	ctx := context.Background()
	require.NoError(t, combinedGradeRepository.Save(ctx, &entities.CombinedGrade{Symbol: "2330", Category: "twStock", TradingDay: "2026-10-02", Grade: "neutral", UpdatedAt: gradedAt}))
	require.NoError(t, combinedGradeRepository.Save(ctx, &entities.CombinedGrade{Symbol: "2330", Category: "twStock", TradingDay: "2026-10-05", Grade: "neutral", UpdatedAt: gradedAt}))

	require.NoError(t, combinedGradeRepository.DeleteExceptTradingDay(ctx, "2026-10-05"))

	remaining, err := combinedGradeRepository.FindAll(ctx, "", "")
	require.NoError(t, err)
	require.Len(t, remaining, 1)
	assert.Equal(t, "2026-10-05", remaining[0].TradingDay)
}

func TestSessionRunRepository_RecordsEachTradingSessionOnce(t *testing.T) {
	database := openTestDatabase(t)
	sessionRunRepository := persistence.NewSessionRunRepository(database)
	ctx := context.Background()
	sessionRun := entities.SessionRun{TradingDay: "2026-10-07", Session: "preMarket", Status: "running", FailedSymbols: []entities.SessionRunFailedSymbol{}, StartedAt: gradedAt}
	require.NoError(t, sessionRunRepository.Create(ctx, &sessionRun))

	duplicateError := sessionRunRepository.Create(ctx, &entities.SessionRun{TradingDay: "2026-10-07", Session: "preMarket", Status: "running", FailedSymbols: []entities.SessionRunFailedSymbol{}, StartedAt: gradedAt})
	otherSessionError := sessionRunRepository.Create(ctx, &entities.SessionRun{TradingDay: "2026-10-07", Session: "intraday", Status: "running", FailedSymbols: []entities.SessionRunFailedSymbol{}, StartedAt: gradedAt})

	assert.ErrorIs(t, duplicateError, service.ErrSessionRunAlreadyExists)
	assert.NoError(t, otherSessionError)
	sessionRun.Status = "succeeded"
	sessionRun.FailedSymbolCount = 1
	sessionRun.FailedSymbols = []entities.SessionRunFailedSymbol{{Symbol: "2330", Category: "twStock", FailureReason: "AI 服務暫時無法使用"}}
	sessionRun.InputTokens = 307
	assert.NoError(t, sessionRunRepository.Update(ctx, &sessionRun))
	storedSessionRun := entities.SessionRun{}
	require.NoError(t, database.First(&storedSessionRun, sessionRun.ID).Error)
	assert.Equal(t, sessionRun.FailedSymbols, storedSessionRun.FailedSymbols)
	assert.Equal(t, int64(307), storedSessionRun.InputTokens)
}

func TestSessionRunRepository_FailAllRunningLeavesFinishedRunsAlone(t *testing.T) {
	database := openTestDatabase(t)
	sessionRunRepository := persistence.NewSessionRunRepository(database)
	ctx := context.Background()
	require.NoError(t, sessionRunRepository.Create(ctx, &entities.SessionRun{TradingDay: "2026-10-07", Session: "preMarket", Status: "running", FailedSymbols: []entities.SessionRunFailedSymbol{}, StartedAt: gradedAt}))
	require.NoError(t, sessionRunRepository.Create(ctx, &entities.SessionRun{TradingDay: "2026-10-06", Session: "afterMarket", Status: "succeeded", FailedSymbols: []entities.SessionRunFailedSymbol{}, StartedAt: gradedAt}))

	require.NoError(t, sessionRunRepository.FailAllRunning(ctx, "服務重新啟動，執行中斷", gradedAt.Add(time.Hour)))

	sessionRuns := []entities.SessionRun{}
	require.NoError(t, database.Order(clause.OrderByColumn{Column: clause.Column{Name: "trading_day"}}).Find(&sessionRuns).Error)
	assert.Equal(t, "succeeded", sessionRuns[0].Status)
	assert.Equal(t, "failed", sessionRuns[1].Status)
	assert.Equal(t, "服務重新啟動，執行中斷", sessionRuns[1].FailureReason)
}

func TestSessionRunRepository_ReportsStorageFailuresApartFromExistingRuns(t *testing.T) {
	database := openTestDatabase(t)
	sessionRunRepository := persistence.NewSessionRunRepository(database)
	connection, err := database.DB()
	require.NoError(t, err)
	require.NoError(t, connection.Close())

	createError := sessionRunRepository.Create(context.Background(), &entities.SessionRun{TradingDay: "2026-10-07", Session: "preMarket", Status: "running", FailedSymbols: []entities.SessionRunFailedSymbol{}, StartedAt: gradedAt})

	assert.Error(t, createError)
	assert.NotErrorIs(t, createError, service.ErrSessionRunAlreadyExists)
}

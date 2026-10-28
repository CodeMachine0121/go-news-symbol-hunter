package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/persistence"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var backtestTaipei = time.FixedZone("Asia/Taipei", 8*60*60)

// event and record ids follow creation order in freshly created tables
func createAnalysisWithStatus(t *testing.T, database *gorm.DB, status string, symbol string, category string, completedAt time.Time) {
	analysisEvent := entities.AnalysisEvent{Symbol: symbol, Category: category, Status: status, StartedAt: completedAt}
	require.NoError(t, persistence.NewAnalysisEventRepository(database).Create(context.Background(), &analysisEvent))
	require.NoError(t, persistence.NewGradeRecordRepository(database).Create(context.Background(), &entities.GradeRecord{
		Source: "manual", AnalysisEventID: &analysisEvent.ID, Symbol: symbol, Category: category, Grade: "bullish", TimeHorizon: "short", Reason: "r",
		KeyEvents: []entities.AnalysisKeyEvent{}, RiskFactors: []string{}, Evidence: []entities.AnalysisEvidence{}, CreatedAt: completedAt,
	}))
}

func createCompletedAnalysis(t *testing.T, database *gorm.DB, symbol string, category string, completedAt time.Time) {
	createAnalysisWithStatus(t, database, "succeeded", symbol, category, completedAt)
}

func createBackgroundRecord(t *testing.T, database *gorm.DB, symbol string, tradingDay string, grade string, createdAt time.Time) {
	require.NoError(t, persistence.NewGradeRecordRepository(database).Create(context.Background(), &entities.GradeRecord{
		Source: "background", Symbol: symbol, Category: "twStock", TradingDay: tradingDay, Grade: grade, TimeHorizon: "short",
		KeyEvents: []entities.AnalysisKeyEvent{}, RiskFactors: []string{}, Evidence: []entities.AnalysisEvidence{}, CreatedAt: createdAt,
	}))
}

func gradeRecordIDsOf(gradeRecords []entities.GradeRecord) []uint {
	gradeRecordIDs := []uint{}
	for _, gradeRecord := range gradeRecords {
		gradeRecordIDs = append(gradeRecordIDs, gradeRecord.ID)
	}
	return gradeRecordIDs
}

func TestGradeRecordRepository_FindForBacktestFiltersByCompletedPeriodAndSymbol(t *testing.T) {
	database := openTestDatabase(t)
	gradeRecordRepository := persistence.NewGradeRecordRepository(database)
	createCompletedAnalysis(t, database, "2330", "twStock", time.Date(2026, 9, 30, 23, 59, 0, 0, backtestTaipei))
	createCompletedAnalysis(t, database, "2330", "twStock", time.Date(2026, 10, 1, 0, 0, 0, 0, backtestTaipei))
	createCompletedAnalysis(t, database, "2317", "twStock", time.Date(2026, 10, 15, 9, 0, 0, 0, backtestTaipei))
	createCompletedAnalysis(t, database, "2330", "usStock", time.Date(2026, 10, 20, 9, 0, 0, 0, backtestTaipei))
	createCompletedAnalysis(t, database, "2330", "twStock", time.Date(2026, 10, 31, 23, 59, 0, 0, backtestTaipei))
	createCompletedAnalysis(t, database, "2330", "twStock", time.Date(2026, 11, 1, 0, 0, 0, 0, backtestTaipei))
	october, err := vo.NewAnalysisCompletedPeriodVo("2026-10-01", "2026-10-31")
	require.NoError(t, err)
	testCases := []struct {
		name                   string
		gradeBacktestFilter    vo.GradeBacktestFilterVo
		expectedGradeRecordIDs []uint
	}{
		{name: "everything oldest first", gradeBacktestFilter: vo.GradeBacktestFilterVo{}, expectedGradeRecordIDs: []uint{1, 2, 3, 4, 5, 6}},
		{name: "both ends of the period are included", gradeBacktestFilter: vo.GradeBacktestFilterVo{CompletedPeriod: october}, expectedGradeRecordIDs: []uint{2, 3, 4, 5}},
		{name: "one symbol in one market", gradeBacktestFilter: vo.GradeBacktestFilterVo{CompletedPeriod: october, SymbolFilterVo: vo.SymbolFilterVo{Symbol: "2330", Category: "twStock"}}, expectedGradeRecordIDs: []uint{2, 5}},
		{name: "only a start", gradeBacktestFilter: vo.GradeBacktestFilterVo{CompletedPeriod: vo.AnalysisCompletedPeriodVo{From: october.From}}, expectedGradeRecordIDs: []uint{2, 3, 4, 5, 6}},
		{name: "only an end", gradeBacktestFilter: vo.GradeBacktestFilterVo{CompletedPeriod: vo.AnalysisCompletedPeriodVo{Before: october.Before}}, expectedGradeRecordIDs: []uint{1, 2, 3, 4, 5}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			gradeRecords, err := gradeRecordRepository.FindForBacktest(context.Background(), testCase.gradeBacktestFilter, 10)

			require.NoError(t, err)
			assert.Equal(t, testCase.expectedGradeRecordIDs, gradeRecordIDsOf(gradeRecords))
		})
	}
}

func TestGradeRecordRepository_FindForBacktestStopsAtTheLimit(t *testing.T) {
	database := openTestDatabase(t)
	gradeRecordRepository := persistence.NewGradeRecordRepository(database)
	for day := 1; day <= 3; day++ {
		createCompletedAnalysis(t, database, "BTC", "crypto", time.Date(2026, 10, day, 0, 0, 0, 0, time.UTC))
	}

	gradeRecords, err := gradeRecordRepository.FindForBacktest(context.Background(), vo.GradeBacktestFilterVo{}, 2)

	require.NoError(t, err)
	assert.Equal(t, []uint{1, 2}, gradeRecordIDsOf(gradeRecords))
}

func TestGradeRecordRepository_FindForBacktestFailsWithoutTheDatabase(t *testing.T) {
	database := openTestDatabase(t)
	sqlDatabase, err := database.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDatabase.Close())

	_, findError := persistence.NewGradeRecordRepository(database).FindForBacktest(context.Background(), vo.GradeBacktestFilterVo{}, 10)

	assert.Error(t, findError)
}

func TestGradeRecordRepository_FindForBacktestOnlyCoversSucceededManualAnalysesButEveryBackgroundRecord(t *testing.T) {
	database := openTestDatabase(t)
	completedAt := time.Date(2026, 10, 7, 2, 15, 0, 0, time.UTC)
	createAnalysisWithStatus(t, database, "succeeded", "2330", "twStock", completedAt)
	// its record was saved but marking the event succeeded failed, and a restart then failed it
	createAnalysisWithStatus(t, database, "failed", "2317", "twStock", completedAt)
	createAnalysisWithStatus(t, database, "running", "BTC", "crypto", completedAt)
	createBackgroundRecord(t, database, "4958", "2026-10-07", "bullish", completedAt.Add(time.Minute))

	gradeRecords, err := persistence.NewGradeRecordRepository(database).FindForBacktest(context.Background(), vo.GradeBacktestFilterVo{}, 10)

	require.NoError(t, err)
	assert.Equal(t, []uint{1, 4}, gradeRecordIDsOf(gradeRecords))
}

func TestGradeRecordRepository_FindForBacktestFiltersBySource(t *testing.T) {
	database := openTestDatabase(t)
	completedAt := time.Date(2026, 10, 7, 2, 15, 0, 0, time.UTC)
	createCompletedAnalysis(t, database, "2330", "twStock", completedAt)
	createBackgroundRecord(t, database, "2330", "2026-10-07", "bullish", completedAt.Add(time.Minute))
	createBackgroundRecord(t, database, "2330", "2026-10-07", "neutral", completedAt.Add(time.Hour))
	testCases := []struct {
		source                 string
		expectedGradeRecordIDs []uint
	}{
		{source: "", expectedGradeRecordIDs: []uint{1, 2, 3}},
		{source: "manual", expectedGradeRecordIDs: []uint{1}},
		{source: "background", expectedGradeRecordIDs: []uint{2, 3}},
	}
	for _, testCase := range testCases {
		t.Run("source "+testCase.source, func(t *testing.T) {
			gradeRecords, err := persistence.NewGradeRecordRepository(database).FindForBacktest(context.Background(), vo.GradeBacktestFilterVo{Source: testCase.source}, 10)

			require.NoError(t, err)
			assert.Equal(t, testCase.expectedGradeRecordIDs, gradeRecordIDsOf(gradeRecords))
		})
	}
}

func TestGradeRecordRepository_FindLatestBackgroundTakesTheNewestRecordOfTheNewestTradingDay(t *testing.T) {
	database := openTestDatabase(t)
	wednesday := time.Date(2026, 10, 7, 0, 0, 0, 0, backtestTaipei)
	createBackgroundRecord(t, database, "2330", "2026-10-06", "bearish", wednesday.Add(-4*time.Hour))
	createBackgroundRecord(t, database, "2330", "2026-10-07", "bullish", wednesday.Add(8*time.Hour+30*time.Minute))
	createBackgroundRecord(t, database, "2330", "2026-10-07", "neutral", wednesday.Add(12*time.Hour+45*time.Minute))
	createBackgroundRecord(t, database, "2317", "2026-10-06", "bearish", wednesday.Add(-4*time.Hour))
	createCompletedAnalysis(t, database, "2330", "twStock", wednesday.Add(20*time.Hour))
	// a symbol with only a user-started analysis has no combined grade
	createCompletedAnalysis(t, database, "4958", "twStock", wednesday.Add(20*time.Hour))
	twStock := vo.MarketCategoryVo{Value: "twStock"}

	gradeRecords, err := persistence.NewGradeRecordRepository(database).FindLatestBackground(context.Background(), []vo.SymbolVo{
		{Value: "2330", Category: twStock}, {Value: "2317", Category: twStock}, {Value: "4958", Category: twStock},
	})

	require.NoError(t, err)
	require.Len(t, gradeRecords, 2)
	assert.Equal(t, "2330", gradeRecords[0].Symbol)
	assert.Equal(t, "2026-10-07", gradeRecords[0].TradingDay)
	assert.Equal(t, "neutral", gradeRecords[0].Grade)
	assert.Equal(t, "2317", gradeRecords[1].Symbol)
	assert.Equal(t, "2026-10-06", gradeRecords[1].TradingDay)
}

func TestGradeRecordRepository_FindLatestBackgroundFailsWithoutTheDatabase(t *testing.T) {
	database := openTestDatabase(t)
	sqlDatabase, err := database.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDatabase.Close())

	_, findError := persistence.NewGradeRecordRepository(database).FindLatestBackground(context.Background(), []vo.SymbolVo{{Value: "2330", Category: vo.MarketCategoryVo{Value: "twStock"}}})

	assert.Error(t, findError)
}

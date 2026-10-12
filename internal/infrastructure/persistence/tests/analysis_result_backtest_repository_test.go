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

// event ids follow creation order in a freshly created table
func createAnalysisWithStatus(t *testing.T, database *gorm.DB, status string, symbol string, category string, completedAt time.Time) {
	analysisEvent := entities.AnalysisEvent{Symbol: symbol, Category: category, Status: status, StartedAt: completedAt}
	require.NoError(t, persistence.NewAnalysisEventRepository(database).Create(context.Background(), &analysisEvent))
	require.NoError(t, persistence.NewAnalysisResultRepository(database).Create(context.Background(), &entities.AnalysisResult{
		AnalysisEventID: analysisEvent.ID, Symbol: symbol, Category: category, Grade: "bullish", TimeHorizon: "short", Reason: "r",
		KeyEvents: []entities.AnalysisKeyEvent{}, RiskFactors: []string{}, Evidence: []entities.AnalysisEvidence{}, CreatedAt: completedAt,
	}))
}

func createCompletedAnalysis(t *testing.T, database *gorm.DB, symbol string, category string, completedAt time.Time) {
	createAnalysisWithStatus(t, database, "succeeded", symbol, category, completedAt)
}

func analysisEventIDsOf(analysisResults []entities.AnalysisResult) []uint {
	analysisEventIDs := []uint{}
	for _, analysisResult := range analysisResults {
		analysisEventIDs = append(analysisEventIDs, analysisResult.AnalysisEventID)
	}
	return analysisEventIDs
}

func TestAnalysisResultRepository_FindForBacktestFiltersByCompletedPeriodAndSymbol(t *testing.T) {
	database := openTestDatabase(t)
	analysisResultRepository := persistence.NewAnalysisResultRepository(database)
	createCompletedAnalysis(t, database, "2330", "twStock", time.Date(2026, 9, 30, 23, 59, 0, 0, backtestTaipei))
	createCompletedAnalysis(t, database, "2330", "twStock", time.Date(2026, 10, 1, 0, 0, 0, 0, backtestTaipei))
	createCompletedAnalysis(t, database, "2317", "twStock", time.Date(2026, 10, 15, 9, 0, 0, 0, backtestTaipei))
	createCompletedAnalysis(t, database, "2330", "usStock", time.Date(2026, 10, 20, 9, 0, 0, 0, backtestTaipei))
	createCompletedAnalysis(t, database, "2330", "twStock", time.Date(2026, 10, 31, 23, 59, 0, 0, backtestTaipei))
	createCompletedAnalysis(t, database, "2330", "twStock", time.Date(2026, 11, 1, 0, 0, 0, 0, backtestTaipei))
	october, err := vo.NewAnalysisCompletedPeriodVo("2026-10-01", "2026-10-31")
	require.NoError(t, err)
	testCases := []struct {
		name                     string
		gradeBacktestFilter      vo.GradeBacktestFilterVo
		expectedAnalysisEventIDs []uint
	}{
		{name: "everything oldest first", gradeBacktestFilter: vo.GradeBacktestFilterVo{}, expectedAnalysisEventIDs: []uint{1, 2, 3, 4, 5, 6}},
		{name: "both ends of the period are included", gradeBacktestFilter: vo.GradeBacktestFilterVo{CompletedPeriod: october}, expectedAnalysisEventIDs: []uint{2, 3, 4, 5}},
		{name: "one symbol in one market", gradeBacktestFilter: vo.GradeBacktestFilterVo{CompletedPeriod: october, SymbolFilterVo: vo.SymbolFilterVo{Symbol: "2330", Category: "twStock"}}, expectedAnalysisEventIDs: []uint{2, 5}},
		{name: "only a start", gradeBacktestFilter: vo.GradeBacktestFilterVo{CompletedPeriod: vo.AnalysisCompletedPeriodVo{From: october.From}}, expectedAnalysisEventIDs: []uint{2, 3, 4, 5, 6}},
		{name: "only an end", gradeBacktestFilter: vo.GradeBacktestFilterVo{CompletedPeriod: vo.AnalysisCompletedPeriodVo{Before: october.Before}}, expectedAnalysisEventIDs: []uint{1, 2, 3, 4, 5}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			analysisResults, err := analysisResultRepository.FindForBacktest(context.Background(), testCase.gradeBacktestFilter, 10)

			require.NoError(t, err)
			assert.Equal(t, testCase.expectedAnalysisEventIDs, analysisEventIDsOf(analysisResults))
		})
	}
}

func TestAnalysisResultRepository_FindForBacktestStopsAtTheLimit(t *testing.T) {
	database := openTestDatabase(t)
	analysisResultRepository := persistence.NewAnalysisResultRepository(database)
	for day := 1; day <= 3; day++ {
		createCompletedAnalysis(t, database, "BTC", "crypto", time.Date(2026, 10, day, 0, 0, 0, 0, time.UTC))
	}

	analysisResults, err := analysisResultRepository.FindForBacktest(context.Background(), vo.GradeBacktestFilterVo{}, 2)

	require.NoError(t, err)
	assert.Equal(t, []uint{1, 2}, analysisEventIDsOf(analysisResults))
}

func TestAnalysisResultRepository_FindForBacktestFailsWithoutTheDatabase(t *testing.T) {
	database := openTestDatabase(t)
	sqlDatabase, err := database.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDatabase.Close())

	_, findError := persistence.NewAnalysisResultRepository(database).FindForBacktest(context.Background(), vo.GradeBacktestFilterVo{}, 10)

	assert.Error(t, findError)
}

func TestAnalysisResultRepository_FindForBacktestOnlyCoversSucceededAnalyses(t *testing.T) {
	database := openTestDatabase(t)
	completedAt := time.Date(2026, 10, 7, 2, 15, 0, 0, time.UTC)
	createAnalysisWithStatus(t, database, "succeeded", "2330", "twStock", completedAt)
	// its result was saved but marking the event succeeded failed, and a restart then failed it
	createAnalysisWithStatus(t, database, "failed", "2317", "twStock", completedAt)
	createAnalysisWithStatus(t, database, "running", "BTC", "crypto", completedAt)

	analysisResults, err := persistence.NewAnalysisResultRepository(database).FindForBacktest(context.Background(), vo.GradeBacktestFilterVo{}, 10)

	require.NoError(t, err)
	assert.Equal(t, []uint{1}, analysisEventIDsOf(analysisResults))
}

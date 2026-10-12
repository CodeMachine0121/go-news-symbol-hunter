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
)

var backtestTaipei = time.FixedZone("Asia/Taipei", 8*60*60)

func createCompletedAnalysis(t *testing.T, analysisResultRepository *persistence.AnalysisResultRepository, analysisEventID uint, symbol string, category string, completedAt time.Time) {
	require.NoError(t, analysisResultRepository.Create(context.Background(), &entities.AnalysisResult{
		AnalysisEventID: analysisEventID, Symbol: symbol, Category: category, Grade: "bullish", TimeHorizon: "short", Reason: "r",
		KeyEvents: []entities.AnalysisKeyEvent{}, RiskFactors: []string{}, Evidence: []entities.AnalysisEvidence{}, CreatedAt: completedAt,
	}))
}

func analysisEventIDsOf(analysisResults []entities.AnalysisResult) []uint {
	analysisEventIDs := []uint{}
	for _, analysisResult := range analysisResults {
		analysisEventIDs = append(analysisEventIDs, analysisResult.AnalysisEventID)
	}
	return analysisEventIDs
}

func TestAnalysisResultRepository_FindForBacktestFiltersByCompletedPeriodAndSymbol(t *testing.T) {
	analysisResultRepository := persistence.NewAnalysisResultRepository(openTestDatabase(t))
	createCompletedAnalysis(t, analysisResultRepository, 1, "2330", "twStock", time.Date(2026, 9, 30, 23, 59, 0, 0, backtestTaipei))
	createCompletedAnalysis(t, analysisResultRepository, 2, "2330", "twStock", time.Date(2026, 10, 1, 0, 0, 0, 0, backtestTaipei))
	createCompletedAnalysis(t, analysisResultRepository, 3, "2317", "twStock", time.Date(2026, 10, 15, 9, 0, 0, 0, backtestTaipei))
	createCompletedAnalysis(t, analysisResultRepository, 4, "2330", "usStock", time.Date(2026, 10, 20, 9, 0, 0, 0, backtestTaipei))
	createCompletedAnalysis(t, analysisResultRepository, 5, "2330", "twStock", time.Date(2026, 10, 31, 23, 59, 0, 0, backtestTaipei))
	createCompletedAnalysis(t, analysisResultRepository, 6, "2330", "twStock", time.Date(2026, 11, 1, 0, 0, 0, 0, backtestTaipei))
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
	analysisResultRepository := persistence.NewAnalysisResultRepository(openTestDatabase(t))
	for analysisEventID := uint(1); analysisEventID <= 3; analysisEventID++ {
		createCompletedAnalysis(t, analysisResultRepository, analysisEventID, "BTC", "crypto", time.Date(2026, 10, int(analysisEventID), 0, 0, 0, 0, time.UTC))
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

package application_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/application"
	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface/mocks"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var taipei = time.FixedZone("Asia/Taipei", 8*60*60)

// 2026-10-07 is a Wednesday
func wednesdayAt(hour int, minute int) time.Time {
	return time.Date(2026, 10, 7, hour, minute, 0, 0, taipei)
}

type sessionGradeFixture struct {
	sessionGradeApplication *application.SessionGradeApplication
	now                     *time.Time
	listedCompanyProxy      *mocks.MockIListedCompanyProxy
	twStockNewsProxy        *mocks.MockINewsProxy
	analystProxy            *mocks.MockIAnalystProxy
	trackedSymbolRepository *mocks.MockITrackedSymbolRepository
	sessionGradeRepository  *mocks.MockISessionGradeRepository
	combinedGradeRepository *mocks.MockICombinedGradeRepository
	sessionRunRepository    *mocks.MockISessionRunRepository
}

func createSessionGradeFixture(t *testing.T, now time.Time) sessionGradeFixture {
	fixture := sessionGradeFixture{
		now:                     &now,
		listedCompanyProxy:      mocks.NewMockIListedCompanyProxy(t),
		twStockNewsProxy:        createNewsProxy(t, "鉅亨網"),
		analystProxy:            mocks.NewMockIAnalystProxy(t),
		trackedSymbolRepository: mocks.NewMockITrackedSymbolRepository(t),
		sessionGradeRepository:  mocks.NewMockISessionGradeRepository(t),
		combinedGradeRepository: mocks.NewMockICombinedGradeRepository(t),
		sessionRunRepository:    mocks.NewMockISessionRunRepository(t),
	}
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().RunAndReturn(func() time.Time { return *fixture.now }).Maybe()
	symbolResolutionService := service.NewSymbolResolutionService(dto.SymbolDirectoryCatalogDto{TwStock: []interfaces.IListedCompanyProxy{fixture.listedCompanyProxy}})
	newsSearchService := service.NewNewsSearchService(symbolResolutionService, clockProxy, dto.NewsProviderCatalogDto{TwStock: []dto.NewsProviderDto{{NewsProxy: fixture.twStockNewsProxy}}})
	fixture.sessionGradeApplication = application.NewSessionGradeApplication(service.NewSessionGradeService(
		symbolResolutionService,
		service.NewAnalystConsultationService(fixture.analystProxy, newsSearchService),
		fixture.trackedSymbolRepository,
		fixture.sessionGradeRepository,
		fixture.combinedGradeRepository,
		fixture.sessionRunRepository,
		clockProxy,
		vo.NewSessionWeightsVo(0.3, 0.2, 0.5),
	))
	return fixture
}

func (fixture sessionGradeFixture) givenSessionRunStarts(expectedTradingDay string, expectedSession string) {
	fixture.sessionRunRepository.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, sessionRun *entities.SessionRun) error {
		if sessionRun.TradingDay != expectedTradingDay || sessionRun.Session != expectedSession || sessionRun.Status != "running" {
			panic("unexpected session run " + sessionRun.TradingDay + " " + sessionRun.Session + " " + sessionRun.Status)
		}
		sessionRun.ID = 7
		return nil
	}).Once()
}

func (fixture sessionGradeFixture) expectFinishedSessionRun() *entities.SessionRun {
	finishedSessionRun := &entities.SessionRun{}
	fixture.sessionRunRepository.EXPECT().Update(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, sessionRun *entities.SessionRun) error {
		*finishedSessionRun = *sessionRun
		return nil
	}).Once()
	return finishedSessionRun
}

func (fixture sessionGradeFixture) givenTrackedSymbols(symbols ...string) {
	trackedSymbols := []entities.TrackedSymbol{}
	for _, symbol := range symbols {
		trackedSymbols = append(trackedSymbols, entities.TrackedSymbol{Symbol: symbol, Category: "twStock", IsTracking: true})
	}
	fixture.trackedSymbolRepository.EXPECT().FindTracking(mock.Anything, "twStock").Return(trackedSymbols, nil).Once()
}

func (fixture sessionGradeFixture) givenCompany(symbol string, companyShortName string) {
	fixture.listedCompanyProxy.EXPECT().FindCompanyShortName(mock.Anything, symbol).Return(companyShortName, true, nil)
}

// the analyst searches the symbol once, then concludes citing the first news it read
func (fixture sessionGradeFixture) givenAnalystConcludes(symbol string, grade string, confidence int) {
	fixture.analystProxy.EXPECT().Respond(mock.Anything, mock.MatchedBy(func(request vo.AnalystRequestVo) bool { return request.Symbol == symbol }), mock.MatchedBy(func(exchanges []vo.AnalystExchangeVo) bool { return len(exchanges) == 0 })).
		Return(vo.AnalystTurnVo{Reply: `{}`, NewsSearches: []vo.AnalystNewsSearchVo{{ToolCallID: "toolu_" + symbol, Symbol: symbol, Category: "twStock"}}, ModelName: "claude-sonnet-5-5", Usage: vo.AnalystUsageVo{InputTokens: 100, OutputTokens: 10}}, nil).Once()
	fixture.analystProxy.EXPECT().Respond(mock.Anything, mock.MatchedBy(func(request vo.AnalystRequestVo) bool { return request.Symbol == symbol }), mock.MatchedBy(func(exchanges []vo.AnalystExchangeVo) bool { return len(exchanges) == 1 })).
		Return(vo.AnalystTurnVo{Conclusion: &vo.RawAnalysisConclusionVo{Grade: grade, Confidence: confidence, Reason: "法說會釋出樂觀展望", KeyEventLinks: []string{"https://news/" + symbol + "/1"}, RiskFactors: []string{"匯率"}}, ModelName: "claude-sonnet-5-5", Usage: vo.AnalystUsageVo{InputTokens: 200, OutputTokens: 20}}, nil).Once()
}

func (fixture sessionGradeFixture) givenNews(companyShortName string, news ...vo.NewsVo) {
	fixture.twStockNewsProxy.EXPECT().FetchNews(mock.Anything, companyShortName).Return(news, nil)
}

// stores session grades in memory so the combined grade sees what was saved
func (fixture sessionGradeFixture) storeSessionGrades(existing ...entities.SessionGrade) *[]entities.SessionGrade {
	stored := append([]entities.SessionGrade{}, existing...)
	fixture.sessionGradeRepository.EXPECT().Save(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, sessionGrade *entities.SessionGrade) error {
		stored = append(stored, *sessionGrade)
		return nil
	}).Maybe()
	fixture.sessionGradeRepository.EXPECT().FindByTradingDay(mock.Anything, mock.Anything, "twStock", "2026-10-07").RunAndReturn(func(_ context.Context, symbol string, _ string, _ string) ([]entities.SessionGrade, error) {
		found := []entities.SessionGrade{}
		for _, sessionGrade := range stored {
			if sessionGrade.Symbol == symbol {
				found = append(found, sessionGrade)
			}
		}
		return found, nil
	}).Maybe()
	return &stored
}

func (fixture sessionGradeFixture) storeCombinedGrades() *[]entities.CombinedGrade {
	stored := []entities.CombinedGrade{}
	fixture.combinedGradeRepository.EXPECT().Save(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, combinedGrade *entities.CombinedGrade) error {
		stored = append(stored, *combinedGrade)
		return nil
	}).Maybe()
	return &stored
}

func TestRunDueTradingSession_PreMarketPurgesOlderDaysThenGradesTrackedTwStocks(t *testing.T) {
	fixture := createSessionGradeFixture(t, wednesdayAt(8, 0))
	fixture.givenSessionRunStarts("2026-10-07", "preMarket")
	fixture.sessionGradeRepository.EXPECT().DeleteExceptTradingDay(mock.Anything, "2026-10-07").Return(nil).Once()
	fixture.combinedGradeRepository.EXPECT().DeleteExceptTradingDay(mock.Anything, "2026-10-07").Return(nil).Once()
	fixture.givenTrackedSymbols("2330")
	fixture.givenCompany("2330", "台積電")
	fixture.givenNews("台積電",
		vo.NewsVo{Title: "台積電法說會", Link: "https://news/2330/1", PublishedAt: time.Date(2026, 10, 6, 20, 0, 0, 0, taipei), ProviderName: "鉅亨網"},
		vo.NewsVo{Title: "開盤後新聞", Link: "https://news/2330/2", PublishedAt: wednesdayAt(9, 10), ProviderName: "鉅亨網"},
	)
	fixture.givenAnalystConcludes("2330", "bullish", 60)
	storedSessionGrades := fixture.storeSessionGrades()
	storedCombinedGrades := fixture.storeCombinedGrades()
	finishedSessionRun := fixture.expectFinishedSessionRun()

	fixture.sessionGradeApplication.RunDueTradingSession(context.Background())

	newsPublishedWindow := fixture.analystProxy.Calls[0].Arguments.Get(1).(vo.AnalystRequestVo).NewsPublishedWindow
	assert.True(t, time.Date(2026, 10, 6, 13, 30, 0, 0, taipei).Equal(newsPublishedWindow.Since), "news since %v", newsPublishedWindow.Since)
	assert.True(t, wednesdayAt(9, 0).Equal(newsPublishedWindow.Before), "news before %v", newsPublishedWindow.Before)
	newsPublishedAt := time.Date(2026, 10, 6, 20, 0, 0, 0, taipei)
	assert.Equal(t, []entities.SessionGrade{{
		Symbol: "2330", Category: "twStock", TradingDay: "2026-10-07", Session: "preMarket",
		Grade: "bullish", Confidence: 60, Reason: "法說會釋出樂觀展望",
		KeyEvents:    []entities.AnalysisKeyEvent{{Title: "台積電法說會", Link: "https://news/2330/1", PublishedAt: newsPublishedAt}},
		RiskFactors:  []string{"匯率"},
		Evidence:     []entities.AnalysisEvidence{{Title: "台積電法說會", Link: "https://news/2330/1", PublishedAt: newsPublishedAt, ProviderName: "鉅亨網"}},
		Model:        "claude-sonnet-5-5",
		InputTokens:  300,
		OutputTokens: 30,
		CreatedAt:    wednesdayAt(8, 0),
	}}, *storedSessionGrades)
	assert.Equal(t, []entities.CombinedGrade{{Symbol: "2330", Category: "twStock", TradingDay: "2026-10-07", Grade: "bullish", CombinedScore: 1, Confidence: 60, UpdatedAt: wednesdayAt(8, 0)}}, *storedCombinedGrades)
	finishedAt := wednesdayAt(8, 0)
	assert.Equal(t, entities.SessionRun{ID: 7, TradingDay: "2026-10-07", Session: "preMarket", Status: "succeeded", SucceededSymbolCount: 1, StartedAt: wednesdayAt(8, 0), FinishedAt: &finishedAt}, *finishedSessionRun)
}

func TestRunDueTradingSession_MondayPreMarketPurgesFridayGrades(t *testing.T) {
	fixture := createSessionGradeFixture(t, time.Date(2026, 10, 5, 8, 0, 0, 0, taipei))
	fixture.givenSessionRunStarts("2026-10-05", "preMarket")
	fixture.sessionGradeRepository.EXPECT().DeleteExceptTradingDay(mock.Anything, "2026-10-05").Return(nil).Once()
	fixture.combinedGradeRepository.EXPECT().DeleteExceptTradingDay(mock.Anything, "2026-10-05").Return(nil).Once()
	fixture.givenTrackedSymbols()
	finishedSessionRun := fixture.expectFinishedSessionRun()

	fixture.sessionGradeApplication.RunDueTradingSession(context.Background())

	assert.Equal(t, "succeeded", finishedSessionRun.Status)
}

func TestRunDueTradingSession_GivesEachSymbolTenMinutesAndAnalyzesOneAtATime(t *testing.T) {
	fixture := createSessionGradeFixture(t, wednesdayAt(12, 30))
	fixture.givenSessionRunStarts("2026-10-07", "intraday")
	fixture.givenTrackedSymbols("2330", "2317")
	fixture.givenCompany("2330", "台積電")
	fixture.givenCompany("2317", "鴻海")
	runningAnalyses := atomic.Int32{}
	overlapped := atomic.Bool{}
	deadlines := make(chan time.Duration, 2)
	fixture.analystProxy.EXPECT().Respond(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, _ vo.AnalystRequestVo, _ []vo.AnalystExchangeVo) (vo.AnalystTurnVo, error) {
		if runningAnalyses.Add(1) > 1 {
			overlapped.Store(true)
		}
		time.Sleep(10 * time.Millisecond)
		runningAnalyses.Add(-1)
		deadline, hasDeadline := ctx.Deadline()
		if hasDeadline {
			deadlines <- time.Until(deadline)
		}
		return vo.AnalystTurnVo{IsRefused: true}, nil
	}).Times(2)
	fixture.expectFinishedSessionRun()

	fixture.sessionGradeApplication.RunDueTradingSession(context.Background())

	assert.False(t, overlapped.Load())
	require.Len(t, deadlines, 2)
	for range 2 {
		remaining := <-deadlines
		assert.True(t, remaining > 9*time.Minute && remaining <= 10*time.Minute, "remaining %v", remaining)
	}
}

func TestRunDueTradingSession_IntradayKeepsOlderDaysAndReadsOnlyIntradayNews(t *testing.T) {
	fixture := createSessionGradeFixture(t, wednesdayAt(12, 30))
	fixture.givenSessionRunStarts("2026-10-07", "intraday")
	fixture.givenTrackedSymbols("2330")
	fixture.givenCompany("2330", "台積電")
	fixture.givenNews("台積電",
		vo.NewsVo{Title: "新聞 A", Link: "https://news/2330/1", PublishedAt: wednesdayAt(10, 15), ProviderName: "鉅亨網"},
		vo.NewsVo{Title: "新聞 B", Link: "https://news/2330/2", PublishedAt: time.Date(2026, 10, 6, 21, 0, 0, 0, taipei), ProviderName: "鉅亨網"},
		vo.NewsVo{Title: "新聞 C", Link: "https://news/2330/3", PublishedAt: wednesdayAt(9, 0), ProviderName: "鉅亨網"},
	)
	fixture.givenAnalystConcludes("2330", "bearish", 40)
	storedSessionGrades := fixture.storeSessionGrades(entities.SessionGrade{Symbol: "2330", Category: "twStock", TradingDay: "2026-10-07", Session: "preMarket", Grade: "bullish", Confidence: 60})
	storedCombinedGrades := fixture.storeCombinedGrades()
	finishedSessionRun := fixture.expectFinishedSessionRun()

	fixture.sessionGradeApplication.RunDueTradingSession(context.Background())

	require.Len(t, *storedSessionGrades, 2)
	evidenceTitles := []string{}
	for _, evidence := range (*storedSessionGrades)[1].Evidence {
		evidenceTitles = append(evidenceTitles, evidence.Title)
	}
	assert.Equal(t, []string{"新聞 A", "新聞 C"}, evidenceTitles)
	assert.Equal(t, []entities.CombinedGrade{{Symbol: "2330", Category: "twStock", TradingDay: "2026-10-07", Grade: "neutral", CombinedScore: 0.2, Confidence: 52, UpdatedAt: wednesdayAt(12, 30)}}, *storedCombinedGrades)
	assert.Equal(t, "succeeded", finishedSessionRun.Status)
}

func TestRunDueTradingSession_AnalyzesWithoutNewsAsNeutral(t *testing.T) {
	fixture := createSessionGradeFixture(t, wednesdayAt(12, 30))
	fixture.givenSessionRunStarts("2026-10-07", "intraday")
	fixture.givenTrackedSymbols("2330")
	fixture.givenCompany("2330", "台積電")
	fixture.givenNews("台積電", vo.NewsVo{Title: "昨晚新聞", Link: "https://news/2330/1", PublishedAt: time.Date(2026, 10, 6, 21, 0, 0, 0, taipei)})
	fixture.givenAnalystConcludes("2330", "bullish", 80)
	storedSessionGrades := fixture.storeSessionGrades()
	fixture.storeCombinedGrades()
	fixture.expectFinishedSessionRun()

	fixture.sessionGradeApplication.RunDueTradingSession(context.Background())

	require.Len(t, *storedSessionGrades, 1)
	assert.Equal(t, "neutral", (*storedSessionGrades)[0].Grade)
	assert.Equal(t, 0, (*storedSessionGrades)[0].Confidence)
}

func TestRunDueTradingSession_DoesNothingOutsideSessionWindows(t *testing.T) {
	testCases := []struct {
		name string
		now  time.Time
	}{
		{name: "Wednesday 09:00 sharp", now: wednesdayAt(9, 0)},
		{name: "Saturday 08:30", now: time.Date(2026, 10, 10, 8, 30, 0, 0, taipei)},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := createSessionGradeFixture(t, testCase.now)

			fixture.sessionGradeApplication.RunDueTradingSession(context.Background())
		})
	}
}

func TestRunDueTradingSession_RunsEachSessionOnlyOnce(t *testing.T) {
	fixture := createSessionGradeFixture(t, wednesdayAt(8, 40))
	fixture.sessionRunRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(service.ErrSessionRunAlreadyExists).Once()

	fixture.sessionGradeApplication.RunDueTradingSession(context.Background())
}

func TestRunDueTradingSession_StopsWhenTheSessionRunCannotBeRecorded(t *testing.T) {
	fixture := createSessionGradeFixture(t, wednesdayAt(8, 40))
	fixture.sessionRunRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(errDatabaseDown).Once()

	fixture.sessionGradeApplication.RunDueTradingSession(context.Background())
}

func TestRunDueTradingSession_CompletesWithNothingToAnalyze(t *testing.T) {
	fixture := createSessionGradeFixture(t, wednesdayAt(20, 0))
	fixture.givenSessionRunStarts("2026-10-07", "afterMarket")
	fixture.givenTrackedSymbols()
	finishedSessionRun := fixture.expectFinishedSessionRun()

	fixture.sessionGradeApplication.RunDueTradingSession(context.Background())

	assert.Equal(t, "succeeded", finishedSessionRun.Status)
	assert.Equal(t, 0, finishedSessionRun.SucceededSymbolCount)
	assert.Equal(t, 0, finishedSessionRun.FailedSymbolCount)
}

func TestRunDueTradingSession_OneFailingSymbolDoesNotStopTheOthers(t *testing.T) {
	fixture := createSessionGradeFixture(t, wednesdayAt(8, 0))
	fixture.givenSessionRunStarts("2026-10-07", "preMarket")
	fixture.sessionGradeRepository.EXPECT().DeleteExceptTradingDay(mock.Anything, "2026-10-07").Return(nil)
	fixture.combinedGradeRepository.EXPECT().DeleteExceptTradingDay(mock.Anything, "2026-10-07").Return(nil)
	fixture.givenTrackedSymbols("2330", "2317")
	fixture.givenCompany("2330", "台積電")
	fixture.givenCompany("2317", "鴻海")
	fixture.analystProxy.EXPECT().Respond(mock.Anything, mock.MatchedBy(func(request vo.AnalystRequestVo) bool { return request.Symbol == "2330" }), mock.Anything).Return(vo.AnalystTurnVo{}, errDatabaseDown).Once()
	fixture.givenNews("鴻海", vo.NewsVo{Title: "鴻海", Link: "https://news/2317/1", PublishedAt: wednesdayAt(7, 0)})
	fixture.givenAnalystConcludes("2317", "bullish", 70)
	storedSessionGrades := fixture.storeSessionGrades()
	fixture.storeCombinedGrades()
	finishedSessionRun := fixture.expectFinishedSessionRun()

	fixture.sessionGradeApplication.RunDueTradingSession(context.Background())

	require.Len(t, *storedSessionGrades, 1)
	assert.Equal(t, "2317", (*storedSessionGrades)[0].Symbol)
	assert.Equal(t, "succeeded", finishedSessionRun.Status)
	assert.Equal(t, 1, finishedSessionRun.SucceededSymbolCount)
	assert.Equal(t, 1, finishedSessionRun.FailedSymbolCount)
}

func TestRunDueTradingSession_CountsEveryStepThatFailsForASymbol(t *testing.T) {
	testCases := []struct {
		name  string
		given func(fixture sessionGradeFixture)
	}{
		{name: "symbol not found", given: func(fixture sessionGradeFixture) {
			fixture.listedCompanyProxy.EXPECT().FindCompanyShortName(mock.Anything, "2330").Return("", false, nil)
		}},
		{name: "session grade not saved", given: func(fixture sessionGradeFixture) {
			fixture.givenAnalysisOf2330()
			fixture.sessionGradeRepository.EXPECT().Save(mock.Anything, mock.Anything).Return(errDatabaseDown)
		}},
		{name: "session grades not read back", given: func(fixture sessionGradeFixture) {
			fixture.givenAnalysisOf2330()
			fixture.sessionGradeRepository.EXPECT().Save(mock.Anything, mock.Anything).Return(nil)
			fixture.sessionGradeRepository.EXPECT().FindByTradingDay(mock.Anything, "2330", "twStock", "2026-10-07").Return(nil, errDatabaseDown)
		}},
		{name: "combined grade not saved", given: func(fixture sessionGradeFixture) {
			fixture.givenAnalysisOf2330()
			fixture.storeSessionGrades()
			fixture.combinedGradeRepository.EXPECT().Save(mock.Anything, mock.Anything).Return(errDatabaseDown)
		}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := createSessionGradeFixture(t, wednesdayAt(12, 30))
			fixture.givenSessionRunStarts("2026-10-07", "intraday")
			fixture.givenTrackedSymbols("2330")
			testCase.given(fixture)
			finishedSessionRun := fixture.expectFinishedSessionRun()

			fixture.sessionGradeApplication.RunDueTradingSession(context.Background())

			assert.Equal(t, "succeeded", finishedSessionRun.Status)
			assert.Equal(t, 0, finishedSessionRun.SucceededSymbolCount)
			assert.Equal(t, 1, finishedSessionRun.FailedSymbolCount)
		})
	}
}

func (fixture sessionGradeFixture) givenAnalysisOf2330() {
	fixture.givenCompany("2330", "台積電")
	fixture.givenNews("台積電", vo.NewsVo{Title: "台積電", Link: "https://news/2330/1", PublishedAt: wednesdayAt(10, 0)})
	fixture.givenAnalystConcludes("2330", "bullish", 60)
}

func TestRunDueTradingSession_FailsTheRunWhenItCannotStart(t *testing.T) {
	testCases := []struct {
		name                  string
		now                   time.Time
		given                 func(fixture sessionGradeFixture)
		expectedFailureReason string
	}{
		{name: "older days cannot be purged", now: wednesdayAt(8, 0), given: func(fixture sessionGradeFixture) {
			fixture.givenSessionRunStarts("2026-10-07", "preMarket")
			fixture.sessionGradeRepository.EXPECT().DeleteExceptTradingDay(mock.Anything, "2026-10-07").Return(errDatabaseDown)
			fixture.combinedGradeRepository.EXPECT().DeleteExceptTradingDay(mock.Anything, "2026-10-07").Return(nil)
		}, expectedFailureReason: "前一天結果刪除失敗"},
		{name: "tracked symbols cannot be read", now: wednesdayAt(12, 30), given: func(fixture sessionGradeFixture) {
			fixture.givenSessionRunStarts("2026-10-07", "intraday")
			fixture.trackedSymbolRepository.EXPECT().FindTracking(mock.Anything, "twStock").Return(nil, errDatabaseDown)
		}, expectedFailureReason: "追蹤標的讀取失敗"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := createSessionGradeFixture(t, testCase.now)
			testCase.given(fixture)
			finishedSessionRun := fixture.expectFinishedSessionRun()

			fixture.sessionGradeApplication.RunDueTradingSession(context.Background())

			assert.Equal(t, "failed", finishedSessionRun.Status)
			assert.Equal(t, testCase.expectedFailureReason, finishedSessionRun.FailureReason)
			require.NotNil(t, finishedSessionRun.FinishedAt)
		})
	}
}

func TestRunDueTradingSession_SurvivesAFinishedRunThatCannotBeRecorded(t *testing.T) {
	fixture := createSessionGradeFixture(t, wednesdayAt(20, 0))
	fixture.givenSessionRunStarts("2026-10-07", "afterMarket")
	fixture.givenTrackedSymbols()
	fixture.sessionRunRepository.EXPECT().Update(mock.Anything, mock.Anything).Return(errDatabaseDown).Once()

	fixture.sessionGradeApplication.RunDueTradingSession(context.Background())
}

func TestRunDueTradingSession_SurvivesAPanic(t *testing.T) {
	fixture := createSessionGradeFixture(t, wednesdayAt(20, 0))
	fixture.sessionRunRepository.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(func(context.Context, *entities.SessionRun) error { panic("boom") }).Once()

	assert.NotPanics(t, func() { fixture.sessionGradeApplication.RunDueTradingSession(context.Background()) })
}

func TestFailInterruptedSessionRuns(t *testing.T) {
	fixture := createSessionGradeFixture(t, wednesdayAt(8, 30))
	fixture.sessionRunRepository.EXPECT().FailAllRunning(mock.Anything, "服務重新啟動，執行中斷", wednesdayAt(8, 30)).Return(nil).Once()

	require.NoError(t, fixture.sessionGradeApplication.FailInterruptedSessionRuns(context.Background()))
}

func TestFailInterruptedSessionRuns_ReportsStorageFailures(t *testing.T) {
	fixture := createSessionGradeFixture(t, wednesdayAt(8, 30))
	fixture.sessionRunRepository.EXPECT().FailAllRunning(mock.Anything, mock.Anything, mock.Anything).Return(errDatabaseDown)

	assert.ErrorIs(t, fixture.sessionGradeApplication.FailInterruptedSessionRuns(context.Background()), service.ErrSessionGradeStorageUnavailable)
}

func TestGetTrackedSymbolGrades_ReturnsCombinedGradesWithTheirSessionGrades(t *testing.T) {
	fixture := createSessionGradeFixture(t, wednesdayAt(14, 0))
	fixture.combinedGradeRepository.EXPECT().FindAll(mock.Anything, "", "").Return([]entities.CombinedGrade{{Symbol: "2330", Category: "twStock", TradingDay: "2026-10-07", Grade: "neutral", CombinedScore: 0.2, Confidence: 52, UpdatedAt: wednesdayAt(12, 31)}}, nil)
	fixture.sessionGradeRepository.EXPECT().FindByTradingDay(mock.Anything, "2330", "twStock", "2026-10-07").Return([]entities.SessionGrade{
		{Session: "preMarket", Grade: "bullish", Confidence: 60, Reason: "法說會", KeyEvents: []entities.AnalysisKeyEvent{{Title: "法說會", Link: "https://news/1", PublishedAt: wednesdayAt(7, 0)}}, RiskFactors: []string{"匯率"}, Evidence: []entities.AnalysisEvidence{{Title: "法說會", Link: "https://news/1", PublishedAt: wednesdayAt(7, 0), ProviderName: "鉅亨網"}}, CreatedAt: wednesdayAt(8, 1)},
		{Session: "intraday", Grade: "bearish", Confidence: 40, Reason: "外資賣超", CreatedAt: wednesdayAt(12, 31)},
	}, nil)

	trackedSymbolGrades, err := fixture.sessionGradeApplication.GetTrackedSymbolGrades(context.Background(), dto.GetTrackedSymbolGradesDto{})

	require.NoError(t, err)
	assert.Equal(t, []dto.TrackedSymbolGradeDto{{
		Symbol: "2330", Category: "twStock", TradingDay: "2026-10-07", Grade: "neutral", CombinedScore: 0.2, Confidence: 52, UpdatedAt: wednesdayAt(12, 31),
		SessionGrades: []dto.SessionGradeDto{
			{Session: "preMarket", Grade: "bullish", Confidence: 60, Reason: "法說會", KeyEvents: []dto.AnalysisKeyEventDto{{Title: "法說會", Link: "https://news/1", PublishedAt: wednesdayAt(7, 0)}}, RiskFactors: []string{"匯率"}, Evidence: []dto.AnalysisEvidenceDto{{Title: "法說會", Link: "https://news/1", PublishedAt: wednesdayAt(7, 0), ProviderName: "鉅亨網"}}, CreatedAt: wednesdayAt(8, 1)},
			{Session: "intraday", Grade: "bearish", Confidence: 40, Reason: "外資賣超", KeyEvents: []dto.AnalysisKeyEventDto{}, RiskFactors: []string{}, Evidence: []dto.AnalysisEvidenceDto{}, CreatedAt: wednesdayAt(12, 31)},
		},
	}}, trackedSymbolGrades)
}

func TestGetTrackedSymbolGrades_StillShowsFridayOnSaturday(t *testing.T) {
	fixture := createSessionGradeFixture(t, time.Date(2026, 10, 10, 10, 0, 0, 0, taipei))
	fixture.combinedGradeRepository.EXPECT().FindAll(mock.Anything, "", "").Return([]entities.CombinedGrade{{Symbol: "2330", Category: "twStock", TradingDay: "2026-10-09", Grade: "bullish"}}, nil)
	fixture.sessionGradeRepository.EXPECT().FindByTradingDay(mock.Anything, "2330", "twStock", "2026-10-09").Return([]entities.SessionGrade{{Session: "afterMarket", Grade: "bullish"}}, nil)

	fixture.sessionGradeApplication.RunDueTradingSession(context.Background())
	trackedSymbolGrades, err := fixture.sessionGradeApplication.GetTrackedSymbolGrades(context.Background(), dto.GetTrackedSymbolGradesDto{})

	require.NoError(t, err)
	require.Len(t, trackedSymbolGrades, 1)
	assert.Equal(t, "2026-10-09", trackedSymbolGrades[0].TradingDay)
	assert.Equal(t, "afterMarket", trackedSymbolGrades[0].SessionGrades[0].Session)
}

func TestGetTrackedSymbolGrades_FiltersByANormalizedSymbol(t *testing.T) {
	fixture := createSessionGradeFixture(t, wednesdayAt(14, 0))
	fixture.combinedGradeRepository.EXPECT().FindAll(mock.Anything, "2603", "twStock").Return([]entities.CombinedGrade{}, nil)

	trackedSymbolGrades, err := fixture.sessionGradeApplication.GetTrackedSymbolGrades(context.Background(), dto.GetTrackedSymbolGradesDto{Symbol: " 2603 ", Category: "twStock"})

	require.NoError(t, err)
	assert.Equal(t, []dto.TrackedSymbolGradeDto{}, trackedSymbolGrades)
}

func TestGetTrackedSymbolGrades_RejectsInvalidFilters(t *testing.T) {
	testCases := []struct {
		name          string
		request       dto.GetTrackedSymbolGradesDto
		expectedError error
	}{
		{name: "symbol without category", request: dto.GetTrackedSymbolGradesDto{Symbol: "2330"}, expectedError: service.ErrSymbolAndCategoryRequiredTogether},
		{name: "category without symbol", request: dto.GetTrackedSymbolGradesDto{Category: "twStock"}, expectedError: service.ErrSymbolAndCategoryRequiredTogether},
		{name: "unsupported category", request: dto.GetTrackedSymbolGradesDto{Symbol: "0700", Category: "hk"}, expectedError: service.ErrMarketCategoryUnsupported},
		{name: "blank symbol", request: dto.GetTrackedSymbolGradesDto{Symbol: "  ", Category: "twStock"}, expectedError: service.ErrSymbolRequired},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := createSessionGradeFixture(t, wednesdayAt(14, 0))

			_, err := fixture.sessionGradeApplication.GetTrackedSymbolGrades(context.Background(), testCase.request)

			assert.ErrorIs(t, err, testCase.expectedError)
		})
	}
}

func TestGetTrackedSymbolGrades_ReportsStorageFailures(t *testing.T) {
	testCases := []struct {
		name  string
		given func(fixture sessionGradeFixture)
	}{
		{name: "combined grades unavailable", given: func(fixture sessionGradeFixture) {
			fixture.combinedGradeRepository.EXPECT().FindAll(mock.Anything, "", "").Return(nil, errDatabaseDown)
		}},
		{name: "session grades unavailable", given: func(fixture sessionGradeFixture) {
			fixture.combinedGradeRepository.EXPECT().FindAll(mock.Anything, "", "").Return([]entities.CombinedGrade{{Symbol: "2330", Category: "twStock", TradingDay: "2026-10-07"}}, nil)
			fixture.sessionGradeRepository.EXPECT().FindByTradingDay(mock.Anything, "2330", "twStock", "2026-10-07").Return(nil, errDatabaseDown)
		}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := createSessionGradeFixture(t, wednesdayAt(14, 0))
			testCase.given(fixture)

			_, err := fixture.sessionGradeApplication.GetTrackedSymbolGrades(context.Background(), dto.GetTrackedSymbolGradesDto{})

			assert.ErrorIs(t, err, service.ErrSessionGradeStorageUnavailable)
		})
	}
}

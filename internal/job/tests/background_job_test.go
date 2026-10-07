package job_test

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
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/job"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// a Wednesday 08:00 in Taipei, inside the pre-market window
var preMarketNow = time.Date(2026, 10, 7, 8, 0, 0, 0, time.FixedZone("Asia/Taipei", 8*60*60))

func createSessionGradeApplication(t *testing.T, sessionRunRepository *mocks.MockISessionRunRepository) *application.SessionGradeApplication {
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(preMarketNow).Maybe()
	symbolResolutionService := service.NewSymbolResolutionService(dto.SymbolDirectoryCatalogDto{TwStock: []interfaces.IListedCompanyProxy{}})
	return application.NewSessionGradeApplication(service.NewSessionGradeService(
		symbolResolutionService,
		service.NewAnalystConsultationService(mocks.NewMockIAnalystProxy(t), service.NewNewsSearchService(symbolResolutionService, clockProxy, dto.NewsProviderCatalogDto{})),
		mocks.NewMockITrackedSymbolRepository(t),
		mocks.NewMockISessionGradeRepository(t),
		mocks.NewMockICombinedGradeRepository(t),
		sessionRunRepository,
		clockProxy,
		vo.NewSessionWeightsVo(0, 0, 0),
	))
}

func TestTradingSessionJob_ChecksForADueSessionAtStartAndOnEveryTick(t *testing.T) {
	sessionRunRepository := mocks.NewMockISessionRunRepository(t)
	checks := atomic.Int32{}
	sessionRunRepository.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(func(context.Context, *entities.SessionRun) error {
		checks.Add(1)
		return service.ErrSessionRunAlreadyExists
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	job.NewTradingSessionJob(createSessionGradeApplication(t, sessionRunRepository), 5*time.Millisecond).Start(ctx)

	assert.Eventually(t, func() bool { return checks.Load() >= 3 }, time.Second, time.Millisecond)
	cancel()
	time.Sleep(20 * time.Millisecond)
	stoppedAt := checks.Load()
	time.Sleep(30 * time.Millisecond)
	assert.Equal(t, stoppedAt, checks.Load())
}

func TestTradingSessionJob_DisabledByANonPositiveInterval(t *testing.T) {
	for _, checkInterval := range []time.Duration{0, -time.Second} {
		sessionRunRepository := mocks.NewMockISessionRunRepository(t)

		job.NewTradingSessionJob(createSessionGradeApplication(t, sessionRunRepository), checkInterval).Start(context.Background())

		time.Sleep(20 * time.Millisecond)
		sessionRunRepository.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	}
}

func TestBackgroundJobManager_StartsEveryJob(t *testing.T) {
	firstJob := mocks.NewMockIBackgroundJob(t)
	secondJob := mocks.NewMockIBackgroundJob(t)
	ctx := context.Background()
	firstJob.EXPECT().Start(ctx).Once()
	secondJob.EXPECT().Start(ctx).Once()

	job.NewBackgroundJobManager([]interfaces.IBackgroundJob{firstJob, secondJob}).StartAll(ctx)
}

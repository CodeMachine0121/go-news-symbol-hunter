package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface/mocks"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var errDatabaseDown = errors.New("database down")

// a Wednesday evening in Taipei, inside the after-market window
var afterMarketNow = time.Date(2026, 10, 7, 20, 0, 0, 0, time.FixedZone("Asia/Taipei", 8*60*60))

func TestSessionGradeService_RunDueTradingSession_ReportsWhetherTheRunWasRecorded(t *testing.T) {
	testCases := []struct {
		name          string
		given         func(sessionRunRepository *mocks.MockISessionRunRepository, trackedSymbolRepository *mocks.MockITrackedSymbolRepository)
		expectedError error
	}{
		{name: "the session already ran", given: func(sessionRunRepository *mocks.MockISessionRunRepository, _ *mocks.MockITrackedSymbolRepository) {
			sessionRunRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(service.ErrSessionRunAlreadyExists)
		}},
		{name: "the run cannot be created", given: func(sessionRunRepository *mocks.MockISessionRunRepository, _ *mocks.MockITrackedSymbolRepository) {
			sessionRunRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(errDatabaseDown)
		}, expectedError: service.ErrSessionGradeStorageUnavailable},
		{name: "the finished run cannot be saved", given: func(sessionRunRepository *mocks.MockISessionRunRepository, trackedSymbolRepository *mocks.MockITrackedSymbolRepository) {
			sessionRunRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
			trackedSymbolRepository.EXPECT().FindTracking(mock.Anything, "twStock").Return([]entities.TrackedSymbol{}, nil)
			sessionRunRepository.EXPECT().Update(mock.Anything, mock.Anything).Return(errDatabaseDown)
		}, expectedError: service.ErrSessionGradeStorageUnavailable},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			clockProxy := mocks.NewMockIClockProxy(t)
			clockProxy.EXPECT().Now().Return(afterMarketNow)
			sessionRunRepository := mocks.NewMockISessionRunRepository(t)
			trackedSymbolRepository := mocks.NewMockITrackedSymbolRepository(t)
			testCase.given(sessionRunRepository, trackedSymbolRepository)
			symbolResolutionService := service.NewSymbolResolutionService(dto.SymbolDirectoryCatalogDto{TwStock: []interfaces.IListedCompanyProxy{}})
			sessionGradeService := service.NewSessionGradeService(
				symbolResolutionService,
				service.NewAnalystConsultationService(mocks.NewMockIAnalystProxy(t), service.NewNewsSearchService(symbolResolutionService, clockProxy, dto.NewsProviderCatalogDto{})),
				trackedSymbolRepository,
				mocks.NewMockISessionGradeRepository(t),
				mocks.NewMockICombinedGradeRepository(t),
				sessionRunRepository,
				clockProxy,
				vo.NewSessionWeightsVo(0, 0, 0),
			)

			err := sessionGradeService.RunDueTradingSession(context.Background())

			if testCase.expectedError == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, testCase.expectedError)
			}
		})
	}
}

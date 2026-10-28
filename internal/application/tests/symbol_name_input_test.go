package application_test

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
	"github.com/stretchr/testify/require"
)

var taiwanDirectory = []vo.SymbolListingVo{
	{Symbol: "2330", Name: "台積電"},
	{Symbol: "0050", Name: "元大台灣50"},
	{Symbol: "2882", Name: "國泰金"},
	{Symbol: "00878", Name: "國泰永續高股息"},
}

// the channel closes once the background analysis has begun, so the test does not end under it
func (fixture symbolAnalysisFixture) expectNewAnalysisEvent() (*entities.AnalysisEvent, chan struct{}) {
	createdAnalysisEvent := &entities.AnalysisEvent{}
	fixture.analysisEventRepository.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, analysisEvent *entities.AnalysisEvent) error {
		*createdAnalysisEvent = *analysisEvent
		return nil
	})
	analysisBegun := make(chan struct{})
	fixture.analysisEventRepository.EXPECT().FindByID(mock.Anything, uint(0)).RunAndReturn(func(context.Context, uint) (*entities.AnalysisEvent, error) {
		close(analysisBegun)
		return nil, nil
	})
	return createdAnalysisEvent, analysisBegun
}

func TestStartSymbolAnalysis_AcceptsASymbolOrAName(t *testing.T) {
	testCases := []struct {
		name               string
		category           string
		typedText          string
		given              func(fixture symbolAnalysisFixture)
		expectedSymbol     string
		expectedSymbolName string
	}{
		{name: "Taiwan stock by company short name", category: "twStock", typedText: "台積電", given: func(fixture symbolAnalysisFixture) {
			fixture.listedCompanyProxy.EXPECT().SearchSymbolListings(mock.Anything, "台積電").Return(taiwanDirectory, nil)
		}, expectedSymbol: "2330", expectedSymbolName: "台積電"},
		{name: "listed ETF by symbol", category: "twStock", typedText: "0050", given: func(fixture symbolAnalysisFixture) {
			fixture.listedCompanyProxy.EXPECT().SearchSymbolListings(mock.Anything, "0050").Return(taiwanDirectory, nil)
		}, expectedSymbol: "0050", expectedSymbolName: "元大台灣50"},
		{name: "coin by name", category: "crypto", typedText: "bitcoin", given: func(fixture symbolAnalysisFixture) {
			fixture.cryptocurrencyProxy.EXPECT().SearchSymbolListings(mock.Anything, "BITCOIN").Return([]vo.SymbolListingVo{{Symbol: "BITCOIN", Name: "HarryPotterObamaSonic10Inu", Priority: 500}, {Symbol: "btc", Name: "Bitcoin", Priority: 1}}, nil)
		}, expectedSymbol: "BTC", expectedSymbolName: "Bitcoin"},
		{name: "US stock by a name typed like a symbol", category: "usStock", typedText: "TESLA", given: func(fixture symbolAnalysisFixture) {
			fixture.usStockCompanyProxy.EXPECT().SearchSymbolListings(mock.Anything, "TESLA").Return([]vo.SymbolListingVo{{Symbol: "TSLA", Name: "Tesla, Inc."}}, nil)
		}, expectedSymbol: "TSLA", expectedSymbolName: "Tesla, Inc."},
		{name: "US directory unavailable falls back to the typed text", category: "usStock", typedText: "Tesla", given: func(fixture symbolAnalysisFixture) {
			fixture.usStockCompanyProxy.EXPECT().SearchSymbolListings(mock.Anything, "TESLA").Return(nil, errDatabaseDown)
		}, expectedSymbol: "TESLA", expectedSymbolName: ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := createSymbolAnalysisFixture(t)
			testCase.given(fixture)
			fixture.analysisEventRepository.EXPECT().FindLatestReusable(mock.Anything, testCase.expectedSymbol, testCase.category).Return(nil, nil)
			createdAnalysisEvent, analysisBegun := fixture.expectNewAnalysisEvent()

			startedSymbolAnalysis, err := fixture.symbolAnalysisApplication.StartSymbolAnalysis(context.Background(), dto.StartSymbolAnalysisDto{Symbol: testCase.typedText, Category: testCase.category})

			require.NoError(t, err)
			assert.True(t, startedSymbolAnalysis.IsNew)
			assert.Equal(t, testCase.expectedSymbol, createdAnalysisEvent.Symbol)
			assert.Equal(t, testCase.expectedSymbolName, createdAnalysisEvent.SymbolName)
			assert.Equal(t, testCase.expectedSymbol, startedSymbolAnalysis.AnalysisEvent.Symbol)
			<-analysisBegun
		})
	}
}

func TestStartSymbolAnalysis_RejectsANameItCannotPinDown(t *testing.T) {
	testCases := []struct {
		name               string
		category           string
		typedText          string
		given              func(fixture symbolAnalysisFixture)
		expectedError      error
		expectedCandidates []dto.SymbolCandidateDto
	}{
		{name: "Taiwan name matching several stocks", category: "twStock", typedText: "國泰", given: func(fixture symbolAnalysisFixture) {
			fixture.listedCompanyProxy.EXPECT().SearchSymbolListings(mock.Anything, "國泰").Return(taiwanDirectory, nil)
		}, expectedError: service.ErrSymbolAmbiguous, expectedCandidates: []dto.SymbolCandidateDto{{Symbol: "00878", SymbolName: "國泰永續高股息"}, {Symbol: "2882", SymbolName: "國泰金"}}},
		{name: "unknown Taiwan name", category: "twStock", typedText: "台基電", given: func(fixture symbolAnalysisFixture) {
			fixture.listedCompanyProxy.EXPECT().SearchSymbolListings(mock.Anything, "台基電").Return(taiwanDirectory, nil)
		}, expectedError: service.ErrSymbolNotFound},
		{name: "unknown coin", category: "crypto", typedText: "ZZZQQ", given: func(fixture symbolAnalysisFixture) {
			fixture.cryptocurrencyProxy.EXPECT().SearchSymbolListings(mock.Anything, "ZZZQQ").Return([]vo.SymbolListingVo{}, nil)
		}, expectedError: service.ErrSymbolNotFound},
		{name: "unknown US symbol while the US directory answers", category: "usStock", typedText: "ZZZZ", given: func(fixture symbolAnalysisFixture) {
			fixture.usStockCompanyProxy.EXPECT().SearchSymbolListings(mock.Anything, "ZZZZ").Return([]vo.SymbolListingVo{}, nil)
		}, expectedError: service.ErrSymbolNotFound},
		{name: "coin directory unavailable", category: "crypto", typedText: "Bitcoin", given: func(fixture symbolAnalysisFixture) {
			fixture.cryptocurrencyProxy.EXPECT().SearchSymbolListings(mock.Anything, "BITCOIN").Return(nil, errDatabaseDown)
		}, expectedError: service.ErrNewsProvidersUnavailable},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := createSymbolAnalysisFixture(t)
			testCase.given(fixture)

			_, err := fixture.symbolAnalysisApplication.StartSymbolAnalysis(context.Background(), dto.StartSymbolAnalysisDto{Symbol: testCase.typedText, Category: testCase.category})

			assert.ErrorIs(t, err, testCase.expectedError)
			symbolAmbiguousError, isAmbiguous := errors.AsType[*service.SymbolAmbiguousError](err)
			assert.Equal(t, testCase.expectedCandidates != nil, isAmbiguous)
			if isAmbiguous {
				assert.Equal(t, testCase.expectedCandidates, symbolAmbiguousError.Candidates)
			}
			fixture.analysisEventRepository.AssertNotCalled(t, "FindLatestReusable", mock.Anything, mock.Anything, mock.Anything)
			fixture.analysisEventRepository.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
		})
	}
}

func TestStartSymbolAnalysis_ANameReusesTheAnalysisStartedBySymbol(t *testing.T) {
	fixture := createSymbolAnalysisFixture(t)
	fixture.listedCompanyProxy.EXPECT().SearchSymbolListings(mock.Anything, "台積電").Return(taiwanDirectory, nil)
	fixture.analysisEventRepository.EXPECT().FindLatestReusable(mock.Anything, "2330", "twStock").Return(&entities.AnalysisEvent{ID: 7, Symbol: "2330", Category: "twStock", SymbolName: "台積電", Status: "succeeded", FinishedAt: finishedAgo(2 * time.Hour)}, nil)
	fixture.gradeRecordRepository.EXPECT().FindByAnalysisEventID(mock.Anything, uint(7)).Return(nil, nil)

	startedSymbolAnalysis, err := fixture.symbolAnalysisApplication.StartSymbolAnalysis(context.Background(), dto.StartSymbolAnalysisDto{Symbol: "台積電", Category: "twStock"})

	require.NoError(t, err)
	assert.False(t, startedSymbolAnalysis.IsNew)
	assert.Equal(t, uint(7), startedSymbolAnalysis.AnalysisEvent.AnalysisEventID)
	fixture.analysisEventRepository.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestStartSymbolAnalysis_ASymbolStartsAfreshOnceTheNamedAnalysisIsOutOfWindow(t *testing.T) {
	fixture := createSymbolAnalysisFixture(t)
	fixture.listedCompanyProxy.EXPECT().SearchSymbolListings(mock.Anything, "2330").Return(taiwanDirectory, nil)
	fixture.analysisEventRepository.EXPECT().FindLatestReusable(mock.Anything, "2330", "twStock").Return(&entities.AnalysisEvent{ID: 7, Symbol: "2330", Category: "twStock", SymbolName: "台積電", Status: "succeeded", FinishedAt: finishedAgo(7 * time.Hour)}, nil)
	createdAnalysisEvent, analysisBegun := fixture.expectNewAnalysisEvent()

	startedSymbolAnalysis, err := fixture.symbolAnalysisApplication.StartSymbolAnalysis(context.Background(), dto.StartSymbolAnalysisDto{Symbol: "2330", Category: "twStock"})

	require.NoError(t, err)
	assert.True(t, startedSymbolAnalysis.IsNew)
	assert.Equal(t, "2330", createdAnalysisEvent.Symbol)
	<-analysisBegun
}

func identifyTaiwanStock(t *testing.T, typedText string, twseListings []vo.SymbolListingVo, twseError error, tpexListings []vo.SymbolListingVo) (vo.ResolvedSymbolVo, error) {
	twseDirectory := mocks.NewMockISymbolDirectoryProxy(t)
	twseDirectory.EXPECT().SearchSymbolListings(mock.Anything, mock.Anything).Return(twseListings, twseError)
	tpexDirectory := mocks.NewMockISymbolDirectoryProxy(t)
	tpexDirectory.EXPECT().SearchSymbolListings(mock.Anything, mock.Anything).Return(tpexListings, nil)
	symbolResolutionService := service.NewSymbolResolutionService(dto.SymbolDirectoryCatalogDto{TwStock: []interfaces.ISymbolDirectoryProxy{twseDirectory, tpexDirectory}})
	return symbolResolutionService.IdentifySymbol(context.Background(), dto.ResolveSymbolDto{Symbol: typedText, Category: "twStock"})
}

func TestIdentifySymbol_TrustsOnlyASymbolMatchWhileATaiwanDirectoryIsUnavailable(t *testing.T) {
	otcDirectory := []vo.SymbolListingVo{{Symbol: "6182", Name: "合晶"}, {Symbol: "00679B", Name: "元大美債20年"}}
	testCases := []struct {
		name           string
		typedText      string
		expectedSymbol string
		expectedError  error
	}{
		{name: "symbol listed by the readable directory", typedText: "6182", expectedSymbol: "6182"},
		{name: "name listed by the readable directory", typedText: "合晶", expectedError: service.ErrNewsProvidersUnavailable},
		{name: "name listed by no readable directory", typedText: "台積電", expectedError: service.ErrNewsProvidersUnavailable},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			resolvedSymbol, err := identifyTaiwanStock(t, testCase.typedText, nil, errDatabaseDown, otcDirectory)

			assert.ErrorIs(t, err, testCase.expectedError)
			assert.Equal(t, testCase.expectedSymbol, resolvedSymbol.Symbol.Value)
		})
	}
}

func TestIdentifySymbol_FindsOtcEtfsByNameWhenEveryDirectoryAnswers(t *testing.T) {
	resolvedSymbol, err := identifyTaiwanStock(t, "元大美債20年", taiwanDirectory, nil, []vo.SymbolListingVo{{Symbol: "00679B", Name: "元大美債20年"}})

	require.NoError(t, err)
	assert.Equal(t, "00679B", resolvedSymbol.Symbol.Value)
	assert.Equal(t, "元大美債20年", resolvedSymbol.SearchKeyword)
}

func TestResolveSymbol_SearchingNewsStillTakesOnlyASymbol(t *testing.T) {
	twseDirectory := mocks.NewMockISymbolDirectoryProxy(t)
	twseDirectory.EXPECT().SearchSymbolListings(mock.Anything, "台積電").Return(taiwanDirectory, nil)
	symbolResolutionService := service.NewSymbolResolutionService(dto.SymbolDirectoryCatalogDto{TwStock: []interfaces.ISymbolDirectoryProxy{twseDirectory}})

	_, err := symbolResolutionService.ResolveSymbol(context.Background(), dto.ResolveSymbolDto{Symbol: "台積電", Category: "twStock"})

	assert.ErrorIs(t, err, service.ErrSymbolNotFound)
}

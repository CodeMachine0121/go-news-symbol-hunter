package twse_test

import (
	"context"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/httpfetch"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface/mocks"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/twse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const listedCompanies = `[{"公司代號":"2330","公司簡稱":"台積電"},{"公司代號":" 1101 ","公司簡稱":" 台泥 "}]`

var lookedUpAt = time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

func startListedCompanyServer(t *testing.T, statusCode int, body string) (*httptest.Server, *atomic.Int32) {
	requestCount := &atomic.Int32{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount.Add(1)
		writer.WriteHeader(statusCode)
		_, _ = writer.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server, requestCount
}

func TestTwseListedCompanyProxy_FindsShortNamesByStockCode(t *testing.T) {
	server, _ := startListedCompanyServer(t, http.StatusOK, listedCompanies)
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(lookedUpAt)
	twseListedCompanyProxy := twse.NewTwseListedCompanyProxy(httpfetch.NewHttpBodyReader(server.Client()), clockProxy, server.URL)

	shortName, found, err := twseListedCompanyProxy.FindCompanyShortName(context.Background(), "2330")
	trimmedShortName, trimmedFound, _ := twseListedCompanyProxy.FindCompanyShortName(context.Background(), "1101")
	_, unlistedFound, _ := twseListedCompanyProxy.FindCompanyShortName(context.Background(), "9999")

	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "台積電", shortName)
	assert.True(t, trimmedFound)
	assert.Equal(t, "台泥", trimmedShortName)
	assert.False(t, unlistedFound)
}

func TestTwseListedCompanyProxy_ReusesTheListForTwentyFourHours(t *testing.T) {
	server, requestCount := startListedCompanyServer(t, http.StatusOK, listedCompanies)
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(lookedUpAt).Once()
	clockProxy.EXPECT().Now().Return(lookedUpAt.Add(24*time.Hour - time.Second)).Once()
	clockProxy.EXPECT().Now().Return(lookedUpAt.Add(24 * time.Hour)).Once()
	twseListedCompanyProxy := twse.NewTwseListedCompanyProxy(httpfetch.NewHttpBodyReader(server.Client()), clockProxy, server.URL)

	_, _, _ = twseListedCompanyProxy.FindCompanyShortName(context.Background(), "2330")
	_, _, _ = twseListedCompanyProxy.FindCompanyShortName(context.Background(), "2330")
	assert.Equal(t, int32(1), requestCount.Load())
	_, _, _ = twseListedCompanyProxy.FindCompanyShortName(context.Background(), "2330")

	assert.Equal(t, int32(2), requestCount.Load())
}

func TestTwseListedCompanyProxy_ReportsUnavailableOrMalformedLists(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		statusCode int
		body       string
	}{
		{name: "unavailable", statusCode: http.StatusBadGateway},
		{name: "malformed", statusCode: http.StatusOK, body: "{"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			server, _ := startListedCompanyServer(t, testCase.statusCode, testCase.body)
			clockProxy := mocks.NewMockIClockProxy(t)
			clockProxy.EXPECT().Now().Return(lookedUpAt)

			_, found, err := twse.NewTwseListedCompanyProxy(httpfetch.NewHttpBodyReader(server.Client()), clockProxy, server.URL).FindCompanyShortName(context.Background(), "2330")

			assert.Error(t, err)
			assert.False(t, found)
		})
	}
}

func TestTwseListedCompanyProxy_ServesCachedLookupsWhileARefreshIsInFlight(t *testing.T) {
	releaseRefresh := make(chan struct{})
	requestCount := &atomic.Int32{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if requestCount.Add(1) == 2 {
			<-releaseRefresh
		}
		_, _ = writer.Write([]byte(listedCompanies))
	}))
	defer server.Close()
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(lookedUpAt).Once()
	clockProxy.EXPECT().Now().Return(lookedUpAt.Add(25 * time.Hour)).Once()
	clockProxy.EXPECT().Now().Return(lookedUpAt.Add(time.Hour))
	twseListedCompanyProxy := twse.NewTwseListedCompanyProxy(httpfetch.NewHttpBodyReader(server.Client()), clockProxy, server.URL)
	_, _, _ = twseListedCompanyProxy.FindCompanyShortName(context.Background(), "2330")
	refreshDone := make(chan struct{})
	go func() {
		_, _, _ = twseListedCompanyProxy.FindCompanyShortName(context.Background(), "2330")
		close(refreshDone)
	}()
	time.Sleep(50 * time.Millisecond)

	shortName, found, err := twseListedCompanyProxy.FindCompanyShortName(context.Background(), "2330")

	close(releaseRefresh)
	<-refreshDone
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "台積電", shortName)
}

func TestTwseListedCompanyProxy_TreatsAListWithoutCompaniesAsAFailure(t *testing.T) {
	for _, body := range []string{`[]`, `[{"code":"2330","name":"台積電"}]`} {
		t.Run(body, func(t *testing.T) {
			server, requestCount := startListedCompanyServer(t, http.StatusOK, body)
			clockProxy := mocks.NewMockIClockProxy(t)
			clockProxy.EXPECT().Now().Return(lookedUpAt)
			twseListedCompanyProxy := twse.NewTwseListedCompanyProxy(httpfetch.NewHttpBodyReader(server.Client()), clockProxy, server.URL)

			_, _, firstError := twseListedCompanyProxy.FindCompanyShortName(context.Background(), "2330")
			_, _, secondError := twseListedCompanyProxy.FindCompanyShortName(context.Background(), "2330")

			assert.Error(t, firstError)
			assert.Error(t, secondError)
			assert.Equal(t, int32(2), requestCount.Load())
		})
	}
}

func TestTwseListedCompanyProxy_DownloadsOnceForConcurrentCacheMisses(t *testing.T) {
	releaseDownload := make(chan struct{})
	requestCount := &atomic.Int32{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount.Add(1)
		<-releaseDownload
		_, _ = writer.Write([]byte(listedCompanies))
	}))
	defer server.Close()
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(lookedUpAt)
	twseListedCompanyProxy := twse.NewTwseListedCompanyProxy(httpfetch.NewHttpBodyReader(server.Client()), clockProxy, server.URL)
	var waitGroup sync.WaitGroup
	shortNames := make([]string, 5)
	for index := range shortNames {
		waitGroup.Go(func() {
			shortNames[index], _, _ = twseListedCompanyProxy.FindCompanyShortName(context.Background(), "2330")
		})
	}
	time.Sleep(50 * time.Millisecond)

	close(releaseDownload)
	waitGroup.Wait()

	assert.Equal(t, int32(1), requestCount.Load())
	assert.Equal(t, []string{"台積電", "台積電", "台積電", "台積電", "台積電"}, shortNames)
}

package yahoofinance_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/httpfetch"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/news"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/yahoofinance"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/utilities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const headlineFeed = `<?xml version="1.0"?><rss version="2.0"><channel><item><title>Apple earnings</title><link>https://example.com/apple</link><pubDate>Tue, 06 Oct 2026 14:00:00 +0000</pubDate><description>&lt;p&gt;Record quarter&lt;/p&gt;</description></item></channel></rss>`

func fetchNewsFrom(t *testing.T, symbol string, statusCode int, body string) ([]vo.NewsVo, url.Values, error) {
	receivedQuery := url.Values{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		receivedQuery = request.URL.Query()
		writer.WriteHeader(statusCode)
		_, _ = writer.Write([]byte(body))
	}))
	defer server.Close()
	httpBodyReader := httpfetch.NewHttpBodyReader(server.Client())
	yahooFinanceProxy := yahoofinance.NewYahooFinanceProxy(httpBodyReader, news.NewRssNewsReader(httpBodyReader, utilities.NewRssFeedParser()), yahoofinance.YahooFinanceUrls{HeadlineFeed: server.URL})

	fetchedNews, err := yahooFinanceProxy.FetchNews(context.Background(), symbol)
	return fetchedNews, receivedQuery, err
}

func TestYahooFinanceProxy_FetchesTheSymbolHeadlines(t *testing.T) {
	fetchedNews, receivedQuery, err := fetchNewsFrom(t, "BRK.B", http.StatusOK, headlineFeed)

	require.NoError(t, err)
	assert.Equal(t, url.Values{"s": {"BRK-B"}, "region": {"US"}, "lang": {"en-US"}}, receivedQuery)
	require.Len(t, fetchedNews, 1)
	fetchedNews[0].PublishedAt = fetchedNews[0].PublishedAt.UTC()
	assert.Equal(t, vo.NewsVo{Title: "Apple earnings", Link: "https://example.com/apple", PublishedAt: time.Date(2026, 10, 6, 14, 0, 0, 0, time.UTC), ProviderName: "Yahoo 財經", Summary: "Record quarter"}, fetchedNews[0])
	assert.Equal(t, "Yahoo 財經", yahoofinance.NewYahooFinanceProxy(nil, nil, yahoofinance.YahooFinanceUrls{}).ProviderName())
}

func TestYahooFinanceProxy_NewsFailsOnUnavailableOrMalformedFeeds(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		statusCode int
		body       string
	}{
		{name: "unavailable", statusCode: http.StatusServiceUnavailable},
		{name: "malformed", statusCode: http.StatusOK, body: "<rss><channel><item>"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			fetchedNews, _, err := fetchNewsFrom(t, "AAPL", testCase.statusCode, testCase.body)

			assert.Error(t, err)
			assert.Nil(t, fetchedNews)
		})
	}
}

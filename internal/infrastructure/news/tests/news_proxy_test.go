package news_test

import (
	"context"
	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/httpfetch"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/news"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/utilities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const rssFeed = `<?xml version="1.0"?><rss version="2.0"><channel><item><title>Bitcoin rallies</title><link>https://example.com/bitcoin</link><pubDate>Tue, 06 Oct 2026 14:00:00 +0000</pubDate><description>&lt;p&gt;BTC tops&lt;/p&gt;</description></item></channel></rss>`

var feedPublishedAt = time.Date(2026, 10, 6, 14, 0, 0, 0, time.UTC)

func startServer(t *testing.T, statusCode int, body string, receivedQuery *url.Values) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if receivedQuery != nil {
			*receivedQuery = request.URL.Query()
		}
		writer.WriteHeader(statusCode)
		_, _ = writer.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

func reader(server *httptest.Server) *httpfetch.HttpBodyReader {
	return httpfetch.NewHttpBodyReader(server.Client())
}

func rssReader(server *httptest.Server) *news.RssNewsReader {
	return news.NewRssNewsReader(reader(server), utilities.NewRssFeedParser())
}

func normalized(news []vo.NewsVo) []vo.NewsVo {
	for index := range news {
		news[index].PublishedAt = news[index].PublishedAt.UTC()
	}
	return news
}

func TestCnyesNewsProxy_SearchesByKeywordAndBuildsArticleLinks(t *testing.T) {
	receivedQuery := url.Values{}
	server := startServer(t, http.StatusOK, `{"data":{"items":[{"newsId":6622909,"title":"台積電法說會 &amp; 展望","summary":"重點&#10;摘要","publishAt":1791294157}]}}`, &receivedQuery)
	cnyesNewsProxy := news.NewCnyesNewsProxy(reader(server), server.URL, "https://news.cnyes.com/news/id/")

	fetchedNews, err := cnyesNewsProxy.FetchNews(context.Background(), "台積電")

	require.NoError(t, err)
	assert.Equal(t, "台積電", receivedQuery.Get("q"))
	assert.Equal(t, "鉅亨網", cnyesNewsProxy.ProviderName())
	assert.Equal(t, []vo.NewsVo{{Title: "台積電法說會 & 展望", Link: "https://news.cnyes.com/news/id/6622909", PublishedAt: time.Unix(1791294157, 0).UTC(), ProviderName: "鉅亨網", Summary: "重點\n摘要"}}, fetchedNews)
}

func TestGoogleNewsProxy_SearchesTheLastSevenDaysInTheConfiguredLocale(t *testing.T) {
	receivedQuery := url.Values{}
	server := startServer(t, http.StatusOK, rssFeed, &receivedQuery)
	googleNewsProxy := news.NewGoogleNewsProxy(rssReader(server), server.URL, news.GoogleNewsTraditionalChineseLocale)

	fetchedNews, err := googleNewsProxy.FetchNews(context.Background(), "台積電")

	require.NoError(t, err)
	assert.Equal(t, url.Values{"q": {"台積電 when:7d"}, "hl": {"zh-TW"}, "gl": {"TW"}, "ceid": {"TW:zh-Hant"}}, receivedQuery)
	assert.Equal(t, "Google 新聞", googleNewsProxy.ProviderName())
	assert.Equal(t, []vo.NewsVo{{Title: "Bitcoin rallies", Link: "https://example.com/bitcoin", PublishedAt: feedPublishedAt, ProviderName: "Google 新聞"}}, normalized(fetchedNews))
}

func TestGoogleNewsProxy_EnglishLocale(t *testing.T) {
	receivedQuery := url.Values{}
	server := startServer(t, http.StatusOK, rssFeed, &receivedQuery)

	_, err := news.NewGoogleNewsProxy(rssReader(server), server.URL, news.GoogleNewsEnglishLocale).FetchNews(context.Background(), "AAPL")

	require.NoError(t, err)
	assert.Equal(t, url.Values{"q": {"AAPL when:7d"}, "hl": {"en-US"}, "gl": {"US"}, "ceid": {"US:en"}}, receivedQuery)
}

func TestWholeFeedProxies_ReturnTheEntireFeed(t *testing.T) {
	server := startServer(t, http.StatusOK, rssFeed, nil)
	testCases := []struct {
		name         string
		newsProxy    interfaces.INewsProxy
		providerName string
	}{
		{name: "CoinDesk", newsProxy: news.NewCoinDeskNewsProxy(rssReader(server), server.URL), providerName: "CoinDesk"},
		{name: "Cointelegraph", newsProxy: news.NewCointelegraphNewsProxy(rssReader(server), server.URL), providerName: "Cointelegraph"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fetchedNews, err := testCase.newsProxy.FetchNews(context.Background(), "ignored")

			require.NoError(t, err)
			assert.Equal(t, testCase.providerName, testCase.newsProxy.ProviderName())
			assert.Equal(t, []vo.NewsVo{{Title: "Bitcoin rallies", Link: "https://example.com/bitcoin", PublishedAt: feedPublishedAt, ProviderName: testCase.providerName, Summary: "BTC tops"}}, normalized(fetchedNews))
		})
	}
}

func TestNewsProxies_FailOnUnavailableOrMalformedSources(t *testing.T) {
	unavailableServer := startServer(t, http.StatusServiceUnavailable, "", nil)
	malformedServer := startServer(t, http.StatusOK, "<rss><channel><item>", nil)
	malformedJsonServer := startServer(t, http.StatusOK, "{", nil)
	cnyesErrorServer := startServer(t, http.StatusOK, `{"statusCode":500,"data":null}`, nil)
	cnyesWithoutItemsServer := startServer(t, http.StatusOK, `{"data":{}}`, nil)
	notRssServer := startServer(t, http.StatusOK, `<error>rate limited</error>`, nil)
	testCases := []struct {
		name      string
		fetchNews func() ([]vo.NewsVo, error)
	}{
		{name: "cnyes unavailable", fetchNews: func() ([]vo.NewsVo, error) {
			return news.NewCnyesNewsProxy(reader(unavailableServer), unavailableServer.URL, "x").FetchNews(context.Background(), "台積電")
		}},
		{name: "cnyes malformed", fetchNews: func() ([]vo.NewsVo, error) {
			return news.NewCnyesNewsProxy(reader(malformedJsonServer), malformedJsonServer.URL, "x").FetchNews(context.Background(), "台積電")
		}},
		{name: "cnyes error payload", fetchNews: func() ([]vo.NewsVo, error) {
			return news.NewCnyesNewsProxy(reader(cnyesErrorServer), cnyesErrorServer.URL, "x").FetchNews(context.Background(), "台積電")
		}},
		{name: "cnyes payload without items", fetchNews: func() ([]vo.NewsVo, error) {
			return news.NewCnyesNewsProxy(reader(cnyesWithoutItemsServer), cnyesWithoutItemsServer.URL, "x").FetchNews(context.Background(), "台積電")
		}},
		{name: "google returns a non-rss document", fetchNews: func() ([]vo.NewsVo, error) {
			return news.NewGoogleNewsProxy(rssReader(notRssServer), notRssServer.URL, news.GoogleNewsEnglishLocale).FetchNews(context.Background(), "AAPL")
		}},
		{name: "google unavailable", fetchNews: func() ([]vo.NewsVo, error) {
			return news.NewGoogleNewsProxy(rssReader(unavailableServer), unavailableServer.URL, news.GoogleNewsEnglishLocale).FetchNews(context.Background(), "AAPL")
		}},
		{name: "google malformed", fetchNews: func() ([]vo.NewsVo, error) {
			return news.NewGoogleNewsProxy(rssReader(malformedServer), malformedServer.URL, news.GoogleNewsEnglishLocale).FetchNews(context.Background(), "AAPL")
		}},
		{name: "coindesk unavailable", fetchNews: func() ([]vo.NewsVo, error) {
			return news.NewCoinDeskNewsProxy(rssReader(unavailableServer), unavailableServer.URL).FetchNews(context.Background(), "")
		}},
		{name: "coindesk malformed", fetchNews: func() ([]vo.NewsVo, error) {
			return news.NewCoinDeskNewsProxy(rssReader(malformedServer), malformedServer.URL).FetchNews(context.Background(), "")
		}},
		{name: "cointelegraph unavailable", fetchNews: func() ([]vo.NewsVo, error) {
			return news.NewCointelegraphNewsProxy(rssReader(unavailableServer), unavailableServer.URL).FetchNews(context.Background(), "")
		}},
		{name: "cointelegraph malformed", fetchNews: func() ([]vo.NewsVo, error) {
			return news.NewCointelegraphNewsProxy(rssReader(malformedServer), malformedServer.URL).FetchNews(context.Background(), "")
		}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fetchedNews, err := testCase.fetchNews()

			assert.Error(t, err)
			assert.Nil(t, fetchedNews)
		})
	}
}

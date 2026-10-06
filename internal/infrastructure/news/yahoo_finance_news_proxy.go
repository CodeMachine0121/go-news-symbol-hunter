package news

import (
	"net/url"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/utilities"
)

type YahooFinanceNewsProxy struct {
	httpBodyReader *utilities.HttpBodyReader
	rssFeedParser  *utilities.RssFeedParser
	feedUrl        string
}

func NewYahooFinanceNewsProxy(httpBodyReader *utilities.HttpBodyReader, rssFeedParser *utilities.RssFeedParser, feedUrl string) *YahooFinanceNewsProxy {
	return &YahooFinanceNewsProxy{httpBodyReader: httpBodyReader, rssFeedParser: rssFeedParser, feedUrl: feedUrl}
}

func (yahooFinanceNewsProxy *YahooFinanceNewsProxy) ProviderName() string {
	return YahooFinanceProviderName
}

func (yahooFinanceNewsProxy *YahooFinanceNewsProxy) FetchNews(searchKeyword string) ([]vo.NewsVo, error) {
	query := url.Values{"s": {searchKeyword}, "region": {"US"}, "lang": {"en-US"}}
	feedBody, err := yahooFinanceNewsProxy.httpBodyReader.Read(yahooFinanceNewsProxy.feedUrl + "?" + query.Encode())
	if err != nil {
		return nil, err
	}
	rssItems, err := yahooFinanceNewsProxy.rssFeedParser.Parse(feedBody)
	if err != nil {
		return nil, err
	}
	news := make([]vo.NewsVo, 0, len(rssItems))
	for _, rssItem := range rssItems {
		news = append(news, vo.NewsVo{Title: rssItem.Title, Link: rssItem.Link, PublishedAt: rssItem.PublishedAt, ProviderName: YahooFinanceProviderName, Summary: rssItem.Description})
	}
	return news, nil
}

package news

import (
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/utilities"
)

type CoinDeskNewsProxy struct {
	httpBodyReader *utilities.HttpBodyReader
	rssFeedParser  *utilities.RssFeedParser
	feedUrl        string
}

func NewCoinDeskNewsProxy(httpBodyReader *utilities.HttpBodyReader, rssFeedParser *utilities.RssFeedParser, feedUrl string) *CoinDeskNewsProxy {
	return &CoinDeskNewsProxy{httpBodyReader: httpBodyReader, rssFeedParser: rssFeedParser, feedUrl: feedUrl}
}

func (coinDeskNewsProxy *CoinDeskNewsProxy) ProviderName() string {
	return CoinDeskProviderName
}

func (coinDeskNewsProxy *CoinDeskNewsProxy) FetchNews(_ string) ([]vo.NewsVo, error) {
	feedBody, err := coinDeskNewsProxy.httpBodyReader.Read(coinDeskNewsProxy.feedUrl)
	if err != nil {
		return nil, err
	}
	rssItems, err := coinDeskNewsProxy.rssFeedParser.Parse(feedBody)
	if err != nil {
		return nil, err
	}
	news := make([]vo.NewsVo, 0, len(rssItems))
	for _, rssItem := range rssItems {
		news = append(news, vo.NewsVo{Title: rssItem.Title, Link: rssItem.Link, PublishedAt: rssItem.PublishedAt, ProviderName: CoinDeskProviderName, Summary: rssItem.Description})
	}
	return news, nil
}

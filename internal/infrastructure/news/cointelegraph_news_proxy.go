package news

import (
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/utilities"
)

type CointelegraphNewsProxy struct {
	httpBodyReader *utilities.HttpBodyReader
	rssFeedParser  *utilities.RssFeedParser
	feedUrl        string
}

func NewCointelegraphNewsProxy(httpBodyReader *utilities.HttpBodyReader, rssFeedParser *utilities.RssFeedParser, feedUrl string) *CointelegraphNewsProxy {
	return &CointelegraphNewsProxy{httpBodyReader: httpBodyReader, rssFeedParser: rssFeedParser, feedUrl: feedUrl}
}

func (cointelegraphNewsProxy *CointelegraphNewsProxy) ProviderName() string {
	return CointelegraphProviderName
}

func (cointelegraphNewsProxy *CointelegraphNewsProxy) FetchNews(_ string) ([]vo.NewsVo, error) {
	feedBody, err := cointelegraphNewsProxy.httpBodyReader.Read(cointelegraphNewsProxy.feedUrl)
	if err != nil {
		return nil, err
	}
	rssItems, err := cointelegraphNewsProxy.rssFeedParser.Parse(feedBody)
	if err != nil {
		return nil, err
	}
	news := make([]vo.NewsVo, 0, len(rssItems))
	for _, rssItem := range rssItems {
		news = append(news, vo.NewsVo{Title: rssItem.Title, Link: rssItem.Link, PublishedAt: rssItem.PublishedAt, ProviderName: CointelegraphProviderName, Summary: rssItem.Description})
	}
	return news, nil
}

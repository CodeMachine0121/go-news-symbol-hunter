package news

import (
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/utilities"
)

type RssNewsReader struct {
	httpBodyReader *utilities.HttpBodyReader
	rssFeedParser  *utilities.RssFeedParser
}

func NewRssNewsReader(httpBodyReader *utilities.HttpBodyReader, rssFeedParser *utilities.RssFeedParser) *RssNewsReader {
	return &RssNewsReader{httpBodyReader: httpBodyReader, rssFeedParser: rssFeedParser}
}

func (rssNewsReader *RssNewsReader) ReadNews(feedUrl string, providerName string, includesSummary bool) ([]vo.NewsVo, error) {
	feedBody, err := rssNewsReader.httpBodyReader.Read(feedUrl)
	if err != nil {
		return nil, err
	}
	rssItems, err := rssNewsReader.rssFeedParser.Parse(feedBody)
	if err != nil {
		return nil, err
	}
	news := make([]vo.NewsVo, 0, len(rssItems))
	for _, rssItem := range rssItems {
		summary := ""
		if includesSummary {
			summary = rssItem.Description
		}
		news = append(news, vo.NewsVo{Title: rssItem.Title, Link: rssItem.Link, PublishedAt: rssItem.PublishedAt, ProviderName: providerName, Summary: summary})
	}
	return news, nil
}

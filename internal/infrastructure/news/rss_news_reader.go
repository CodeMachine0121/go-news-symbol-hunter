package news

import (
	"context"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/httpfetch"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/utilities"
)

type RssNewsReader struct {
	httpBodyReader *httpfetch.HttpBodyReader
	rssFeedParser  *utilities.RssFeedParser
}

func NewRssNewsReader(httpBodyReader *httpfetch.HttpBodyReader, rssFeedParser *utilities.RssFeedParser) *RssNewsReader {
	return &RssNewsReader{httpBodyReader: httpBodyReader, rssFeedParser: rssFeedParser}
}

func (rssNewsReader *RssNewsReader) ReadNews(ctx context.Context, feedUrl string, providerName string, includesSummary bool) ([]vo.NewsVo, error) {
	feedBody, err := rssNewsReader.httpBodyReader.Read(ctx, feedUrl)
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

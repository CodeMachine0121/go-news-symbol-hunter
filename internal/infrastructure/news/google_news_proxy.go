package news

import (
	"net/url"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/utilities"
)

type GoogleNewsLocale struct {
	Language string
	Country  string
	Edition  string
}

var (
	GoogleNewsTraditionalChineseLocale = GoogleNewsLocale{Language: "zh-TW", Country: "TW", Edition: "TW:zh-Hant"}
	GoogleNewsEnglishLocale            = GoogleNewsLocale{Language: "en-US", Country: "US", Edition: "US:en"}
)

type GoogleNewsProxy struct {
	httpBodyReader *utilities.HttpBodyReader
	rssFeedParser  *utilities.RssFeedParser
	searchUrl      string
	locale         GoogleNewsLocale
}

func NewGoogleNewsProxy(httpBodyReader *utilities.HttpBodyReader, rssFeedParser *utilities.RssFeedParser, searchUrl string, locale GoogleNewsLocale) *GoogleNewsProxy {
	return &GoogleNewsProxy{httpBodyReader: httpBodyReader, rssFeedParser: rssFeedParser, searchUrl: searchUrl, locale: locale}
}

func (googleNewsProxy *GoogleNewsProxy) ProviderName() string {
	return GoogleNewsProviderName
}

func (googleNewsProxy *GoogleNewsProxy) FetchNews(searchKeyword string) ([]vo.NewsVo, error) {
	query := url.Values{
		"q":    {searchKeyword + " when:7d"},
		"hl":   {googleNewsProxy.locale.Language},
		"gl":   {googleNewsProxy.locale.Country},
		"ceid": {googleNewsProxy.locale.Edition},
	}
	feedBody, err := googleNewsProxy.httpBodyReader.Read(googleNewsProxy.searchUrl + "?" + query.Encode())
	if err != nil {
		return nil, err
	}
	rssItems, err := googleNewsProxy.rssFeedParser.Parse(feedBody)
	if err != nil {
		return nil, err
	}
	news := make([]vo.NewsVo, 0, len(rssItems))
	for _, rssItem := range rssItems {
		// Google News descriptions only repeat the headline and outlet, so they are not a summary
		news = append(news, vo.NewsVo{Title: rssItem.Title, Link: rssItem.Link, PublishedAt: rssItem.PublishedAt, ProviderName: GoogleNewsProviderName})
	}
	return news, nil
}

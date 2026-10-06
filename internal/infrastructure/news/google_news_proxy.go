package news

import (
	"context"
	"net/url"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
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
	rssNewsReader *RssNewsReader
	searchUrl     string
	locale        GoogleNewsLocale
}

func NewGoogleNewsProxy(rssNewsReader *RssNewsReader, searchUrl string, locale GoogleNewsLocale) *GoogleNewsProxy {
	return &GoogleNewsProxy{rssNewsReader: rssNewsReader, searchUrl: searchUrl, locale: locale}
}

func (googleNewsProxy *GoogleNewsProxy) ProviderName() string {
	return GoogleNewsProviderName
}

func (googleNewsProxy *GoogleNewsProxy) FetchNews(ctx context.Context, searchKeyword string) ([]vo.NewsVo, error) {
	query := url.Values{
		"q":    {searchKeyword + " when:7d"},
		"hl":   {googleNewsProxy.locale.Language},
		"gl":   {googleNewsProxy.locale.Country},
		"ceid": {googleNewsProxy.locale.Edition},
	}
	// Google News descriptions only repeat the headline and outlet, so they are not a summary
	return googleNewsProxy.rssNewsReader.ReadNews(ctx, googleNewsProxy.searchUrl+"?"+query.Encode(), GoogleNewsProviderName, false)
}

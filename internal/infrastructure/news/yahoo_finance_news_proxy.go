package news

import (
	"net/url"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

type YahooFinanceNewsProxy struct {
	rssNewsReader *RssNewsReader
	feedUrl       string
}

func NewYahooFinanceNewsProxy(rssNewsReader *RssNewsReader, feedUrl string) *YahooFinanceNewsProxy {
	return &YahooFinanceNewsProxy{rssNewsReader: rssNewsReader, feedUrl: feedUrl}
}

func (yahooFinanceNewsProxy *YahooFinanceNewsProxy) ProviderName() string {
	return YahooFinanceProviderName
}

func (yahooFinanceNewsProxy *YahooFinanceNewsProxy) FetchNews(searchKeyword string) ([]vo.NewsVo, error) {
	feedUrl := yahooFinanceNewsProxy.feedUrl + "?" + url.Values{"s": {searchKeyword}, "region": {"US"}, "lang": {"en-US"}}.Encode()
	return yahooFinanceNewsProxy.rssNewsReader.ReadNews(feedUrl, YahooFinanceProviderName, true)
}

package news

import (
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

type CoinDeskNewsProxy struct {
	rssNewsReader *RssNewsReader
	feedUrl       string
}

func NewCoinDeskNewsProxy(rssNewsReader *RssNewsReader, feedUrl string) *CoinDeskNewsProxy {
	return &CoinDeskNewsProxy{rssNewsReader: rssNewsReader, feedUrl: feedUrl}
}

func (coinDeskNewsProxy *CoinDeskNewsProxy) ProviderName() string {
	return CoinDeskProviderName
}

func (coinDeskNewsProxy *CoinDeskNewsProxy) FetchNews(_ string) ([]vo.NewsVo, error) {
	return coinDeskNewsProxy.rssNewsReader.ReadNews(coinDeskNewsProxy.feedUrl, CoinDeskProviderName, true)
}

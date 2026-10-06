package news

import (
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

type CointelegraphNewsProxy struct {
	rssNewsReader *RssNewsReader
	feedUrl       string
}

func NewCointelegraphNewsProxy(rssNewsReader *RssNewsReader, feedUrl string) *CointelegraphNewsProxy {
	return &CointelegraphNewsProxy{rssNewsReader: rssNewsReader, feedUrl: feedUrl}
}

func (cointelegraphNewsProxy *CointelegraphNewsProxy) ProviderName() string {
	return CointelegraphProviderName
}

func (cointelegraphNewsProxy *CointelegraphNewsProxy) FetchNews(_ string) ([]vo.NewsVo, error) {
	return cointelegraphNewsProxy.rssNewsReader.ReadNews(cointelegraphNewsProxy.feedUrl, CointelegraphProviderName, true)
}

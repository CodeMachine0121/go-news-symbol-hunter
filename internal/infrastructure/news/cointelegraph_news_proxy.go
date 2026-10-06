package news

import (
	"context"
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

func (cointelegraphNewsProxy *CointelegraphNewsProxy) FetchNews(ctx context.Context, _ string) ([]vo.NewsVo, error) {
	return cointelegraphNewsProxy.rssNewsReader.ReadNews(ctx, cointelegraphNewsProxy.feedUrl, CointelegraphProviderName, true)
}

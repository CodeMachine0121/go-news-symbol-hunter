package news

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/httpfetch"
	"html"
	"net/url"
	"strings"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

var errCnyesResponseWithoutItems = errors.New("cnyes response has no news items")

type cnyesSearchResponse struct {
	Data *struct {
		Items *[]cnyesNewsItem `json:"items"`
	} `json:"data"`
}

type cnyesNewsItem struct {
	NewsID    int64  `json:"newsId"`
	Title     string `json:"title"`
	Summary   string `json:"summary"`
	PublishAt int64  `json:"publishAt"`
}

type CnyesNewsProxy struct {
	httpBodyReader *httpfetch.HttpBodyReader
	searchUrl      string
	articleBaseUrl string
}

func NewCnyesNewsProxy(httpBodyReader *httpfetch.HttpBodyReader, searchUrl string, articleBaseUrl string) *CnyesNewsProxy {
	return &CnyesNewsProxy{httpBodyReader: httpBodyReader, searchUrl: searchUrl, articleBaseUrl: articleBaseUrl}
}

func (cnyesNewsProxy *CnyesNewsProxy) ProviderName() string {
	return CnyesProviderName
}

func (cnyesNewsProxy *CnyesNewsProxy) FetchNews(ctx context.Context, searchKeyword string) ([]vo.NewsVo, error) {
	responseBody, err := cnyesNewsProxy.httpBodyReader.Read(ctx, cnyesNewsProxy.searchUrl+"?limit=30&page=1&q="+url.QueryEscape(searchKeyword))
	if err != nil {
		return nil, err
	}
	var searchResponse cnyesSearchResponse
	if err := json.Unmarshal(responseBody, &searchResponse); err != nil {
		return nil, err
	}
	if searchResponse.Data == nil || searchResponse.Data.Items == nil {
		return nil, errCnyesResponseWithoutItems
	}
	news := make([]vo.NewsVo, 0, len(*searchResponse.Data.Items))
	for _, item := range *searchResponse.Data.Items {
		news = append(news, vo.NewsVo{
			Title:        strings.TrimSpace(html.UnescapeString(item.Title)),
			Link:         fmt.Sprintf("%s/%d", strings.TrimRight(cnyesNewsProxy.articleBaseUrl, "/"), item.NewsID),
			PublishedAt:  time.Unix(item.PublishAt, 0).UTC(),
			ProviderName: CnyesProviderName,
			Summary:      strings.TrimSpace(html.UnescapeString(item.Summary)),
		})
	}
	return news, nil
}

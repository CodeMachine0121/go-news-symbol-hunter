package domains_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/stretchr/testify/assert"
)

var searchedAt = time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

func titlesOf(newsDtos []dto.NewsDto) []string {
	titles := []string{}
	for _, newsDto := range newsDtos {
		titles = append(titles, newsDto.Title)
	}
	return titles
}

func TestNewsCollectionDomain_KeepMentioning(t *testing.T) {
	newsCollection := domains.NewNewsCollectionDomain([]vo.NewsVo{
		{Title: "Bitcoin breaks out", PublishedAt: searchedAt},
		{Title: "Markets wrap", Summary: "btc led the rally", PublishedAt: searchedAt},
		{Title: "Ether upgrade ships", Summary: "ETH news", PublishedAt: searchedAt},
	})

	mentioningNews := newsCollection.KeepMentioning([]string{"Bitcoin", "BTC"}).Curate(searchedAt).ToDtos()

	assert.Equal(t, []string{"Bitcoin breaks out", "Markets wrap"}, titlesOf(mentioningNews))
}

func TestNewsCollectionDomain_Curate_KeepsOnlyTheLastSevenDays(t *testing.T) {
	newsCollection := domains.NewNewsCollectionDomain([]vo.NewsVo{
		{Title: "six days twenty-three hours ago", PublishedAt: searchedAt.Add(-(6*24 + 23) * time.Hour)},
		{Title: "seven days one hour ago", PublishedAt: searchedAt.Add(-(7*24 + 1) * time.Hour)},
		{Title: "exactly seven days ago", PublishedAt: searchedAt.Add(-7 * 24 * time.Hour)},
	})

	curatedNews := newsCollection.Curate(searchedAt).ToDtos()

	assert.Equal(t, []string{"six days twenty-three hours ago", "exactly seven days ago"}, titlesOf(curatedNews))
}

func TestNewsCollectionDomain_Curate_KeepsTheNewerOfDuplicateTitles(t *testing.T) {
	cnyesNews := domains.NewNewsCollectionDomain([]vo.NewsVo{{Title: "台積電法說會", ProviderName: "鉅亨網", PublishedAt: searchedAt.Add(-2 * time.Hour)}})
	googleNews := domains.NewNewsCollectionDomain([]vo.NewsVo{{Title: " 台積電法說會 ", ProviderName: "Google 新聞", PublishedAt: searchedAt.Add(-1 * time.Hour)}})

	curatedNews := cnyesNews.Merge(googleNews).Curate(searchedAt).ToDtos()

	assert.Len(t, curatedNews, 1)
	assert.Equal(t, "Google 新聞", curatedNews[0].ProviderName)
}

func TestNewsCollectionDomain_Curate_TreatsTitlesCaseInsensitively(t *testing.T) {
	newsCollection := domains.NewNewsCollectionDomain([]vo.NewsVo{
		{Title: "Apple Earnings", PublishedAt: searchedAt},
		{Title: "apple earnings", PublishedAt: searchedAt.Add(-time.Hour)},
	})

	assert.Len(t, newsCollection.Curate(searchedAt).ToDtos(), 1)
}

func TestNewsCollectionDomain_Curate_SortsNewestFirst(t *testing.T) {
	newsCollection := domains.NewNewsCollectionDomain([]vo.NewsVo{
		{Title: "day before yesterday", PublishedAt: searchedAt.Add(-48 * time.Hour)},
		{Title: "today", PublishedAt: searchedAt},
		{Title: "yesterday", PublishedAt: searchedAt.Add(-24 * time.Hour)},
	})

	assert.Equal(t, []string{"today", "yesterday", "day before yesterday"}, titlesOf(newsCollection.Curate(searchedAt).ToDtos()))
}

func TestNewsCollectionDomain_Curate_ReturnsAtMostThirtyNewest(t *testing.T) {
	news := []vo.NewsVo{}
	for index := range 45 {
		news = append(news, vo.NewsVo{Title: fmt.Sprintf("news %02d", index), PublishedAt: searchedAt.Add(-time.Duration(index) * time.Minute)})
	}

	curatedNews := domains.NewNewsCollectionDomain(news).Curate(searchedAt).ToDtos()

	assert.Len(t, curatedNews, 30)
	assert.Equal(t, "news 00", curatedNews[0].Title)
	assert.Equal(t, "news 29", curatedNews[29].Title)
}

func TestNewsCollectionDomain_ToDtos_CarriesEveryField(t *testing.T) {
	newsCollection := domains.NewNewsCollectionDomain([]vo.NewsVo{{Title: "台積電法說會", Link: "https://news.cnyes.com/news/id/1", PublishedAt: searchedAt, ProviderName: "鉅亨網", Summary: "法說會摘要"}})

	assert.Equal(t, []dto.NewsDto{{Title: "台積電法說會", Link: "https://news.cnyes.com/news/id/1", PublishedAt: searchedAt, ProviderName: "鉅亨網", Summary: "法說會摘要"}}, newsCollection.ToDtos())
}

func TestNewsCollectionDomain_EmptyCollectionYieldsAnEmptyList(t *testing.T) {
	assert.Equal(t, []dto.NewsDto{}, domains.NewNewsCollectionDomain(nil).Curate(searchedAt).ToDtos())
}

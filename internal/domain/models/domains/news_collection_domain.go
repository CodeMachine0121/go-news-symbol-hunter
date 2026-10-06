package domains

import (
	"slices"
	"strings"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

const (
	NewsSearchWindow       = 7 * 24 * time.Hour
	NewsSearchMaximumCount = 30
)

type NewsCollectionDomain struct {
	news []vo.NewsVo
}

func NewNewsCollectionDomain(news []vo.NewsVo) NewsCollectionDomain {
	return NewsCollectionDomain{news: news}
}

func (newsCollectionDomain NewsCollectionDomain) KeepMentioning(relevanceTerms []string) NewsCollectionDomain {
	mentioningNews := []vo.NewsVo{}
	for _, news := range newsCollectionDomain.news {
		searchableText := strings.ToLower(news.Title + " " + news.Summary)
		if slices.ContainsFunc(relevanceTerms, func(relevanceTerm string) bool {
			return strings.Contains(searchableText, strings.ToLower(relevanceTerm))
		}) {
			mentioningNews = append(mentioningNews, news)
		}
	}
	return NewsCollectionDomain{news: mentioningNews}
}

func (newsCollectionDomain NewsCollectionDomain) Merge(otherNewsCollection NewsCollectionDomain) NewsCollectionDomain {
	return NewsCollectionDomain{news: slices.Concat(newsCollectionDomain.news, otherNewsCollection.news)}
}

func (newsCollectionDomain NewsCollectionDomain) Curate(now time.Time) NewsCollectionDomain {
	publishedSince := now.Add(-NewsSearchWindow)
	recentNews := slices.DeleteFunc(slices.Clone(newsCollectionDomain.news), func(news vo.NewsVo) bool {
		return news.PublishedAt.Before(publishedSince)
	})
	slices.SortStableFunc(recentNews, func(firstNews vo.NewsVo, secondNews vo.NewsVo) int {
		return secondNews.PublishedAt.Compare(firstNews.PublishedAt)
	})
	seenTitles := map[string]bool{}
	curatedNews := []vo.NewsVo{}
	for _, news := range recentNews {
		normalizedTitle := strings.ToLower(strings.TrimSpace(news.Title))
		if seenTitles[normalizedTitle] || len(curatedNews) == NewsSearchMaximumCount {
			continue
		}
		seenTitles[normalizedTitle] = true
		curatedNews = append(curatedNews, news)
	}
	return NewsCollectionDomain{news: curatedNews}
}

func (newsCollectionDomain NewsCollectionDomain) ToDtos() []dto.NewsDto {
	newsDtos := make([]dto.NewsDto, 0, len(newsCollectionDomain.news))
	for _, news := range newsCollectionDomain.news {
		newsDtos = append(newsDtos, dto.NewsDto{
			Title:        news.Title,
			Link:         news.Link,
			PublishedAt:  news.PublishedAt,
			ProviderName: news.ProviderName,
			Summary:      news.Summary,
		})
	}
	return newsDtos
}

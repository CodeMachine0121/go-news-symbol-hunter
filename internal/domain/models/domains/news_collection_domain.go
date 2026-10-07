package domains

import (
	"regexp"
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

var asciiWordPattern = regexp.MustCompile(`^[A-Za-z0-9]+$`)

type NewsCollectionDomain struct {
	news []vo.NewsVo
}

func NewNewsCollectionDomain(news []vo.NewsVo) NewsCollectionDomain {
	return NewsCollectionDomain{news: news}
}

func (newsCollectionDomain NewsCollectionDomain) KeepMentioning(relevanceTerms []string) NewsCollectionDomain {
	relevancePatterns := make([]*regexp.Regexp, 0, len(relevanceTerms))
	for _, relevanceTerm := range relevanceTerms {
		// latin terms must match whole words (ETH must not match "something"); CJK has no word boundaries
		wordBoundary := ""
		if asciiWordPattern.MatchString(relevanceTerm) {
			wordBoundary = `\b`
		}
		relevancePatterns = append(relevancePatterns, regexp.MustCompile(`(?i)`+wordBoundary+regexp.QuoteMeta(relevanceTerm)+wordBoundary))
	}
	mentioningNews := []vo.NewsVo{}
	for _, news := range newsCollectionDomain.news {
		searchableText := news.Title + " " + news.Summary
		if slices.ContainsFunc(relevancePatterns, func(relevancePattern *regexp.Regexp) bool {
			return relevancePattern.MatchString(searchableText)
		}) {
			mentioningNews = append(mentioningNews, news)
		}
	}
	return NewsCollectionDomain{news: mentioningNews}
}

func (newsCollectionDomain NewsCollectionDomain) Merge(otherNewsCollection NewsCollectionDomain) NewsCollectionDomain {
	return NewsCollectionDomain{news: slices.Concat(newsCollectionDomain.news, otherNewsCollection.news)}
}

func (newsCollectionDomain NewsCollectionDomain) Curate(now time.Time, newsPublishedWindow vo.NewsPublishedWindowVo) NewsCollectionDomain {
	publishedSince := now.Add(-NewsSearchWindow)
	if newsPublishedWindow.Since.After(publishedSince) {
		publishedSince = newsPublishedWindow.Since
	}
	recentNews := slices.DeleteFunc(slices.Clone(newsCollectionDomain.news), func(news vo.NewsVo) bool {
		return news.PublishedAt.Before(publishedSince) || (!newsPublishedWindow.Before.IsZero() && !news.PublishedAt.Before(newsPublishedWindow.Before))
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

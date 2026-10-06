package utilities

import (
	"encoding/xml"
	"html"
	"regexp"
	"strings"
	"time"
)

var (
	rssPublishedAtLayouts = []string{time.RFC1123Z, time.RFC1123, "Mon, 2 Jan 2006 15:04:05 -0700", "Mon, 2 Jan 2006 15:04:05 MST"}
	htmlTagPattern        = regexp.MustCompile(`<[^>]*>`)
)

type RssItem struct {
	Title       string
	Link        string
	PublishedAt time.Time
	Description string
}

type rssPublishedAt string

type rssText string

type rssSource string

type rssDocument struct {
	XMLName xml.Name `xml:"rss"`
	Channel struct {
		Items []struct {
			Title       rssText        `xml:"title"`
			Link        string         `xml:"link"`
			PubDate     rssPublishedAt `xml:"pubDate"`
			Description rssText        `xml:"description"`
			Source      rssSource      `xml:"source"`
		} `xml:"item"`
	} `xml:"channel"`
}

type RssFeedParser struct{}

func NewRssFeedParser() *RssFeedParser {
	return &RssFeedParser{}
}

func (rssFeedParser *RssFeedParser) Parse(feedBody []byte) ([]RssItem, error) {
	var document rssDocument
	if err := xml.Unmarshal(feedBody, &document); err != nil {
		return nil, err
	}
	rssItems := []RssItem{}
	for _, item := range document.Channel.Items {
		publishedAt, parsed := item.PubDate.toTime()
		if !parsed {
			continue
		}
		rssItems = append(rssItems, RssItem{
			Title:       item.Source.trimFrom(item.Title.toPlainText()),
			Link:        strings.TrimSpace(item.Link),
			PublishedAt: publishedAt,
			Description: item.Description.toPlainText(),
		})
	}
	return rssItems, nil
}

func (publishedAt rssPublishedAt) toTime() (time.Time, bool) {
	for _, layout := range rssPublishedAtLayouts {
		if parsedPublishedAt, err := time.Parse(layout, strings.TrimSpace(string(publishedAt))); err == nil {
			return parsedPublishedAt, true
		}
	}
	return time.Time{}, false
}

func (text rssText) toPlainText() string {
	withoutTags := htmlTagPattern.ReplaceAllString(string(text), " ")
	return strings.Join(strings.Fields(html.UnescapeString(withoutTags)), " ")
}

// aggregators such as Google News append " - <outlet>" to every title
func (source rssSource) trimFrom(title string) string {
	outlet := strings.TrimSpace(string(source))
	if outlet == "" {
		return title
	}
	return strings.TrimSpace(strings.TrimSuffix(title, " - "+outlet))
}

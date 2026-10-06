package utilities_test

import (
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/utilities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRssFeedParser_ParsesItemsAsPlainText(t *testing.T) {
	feedBody := []byte(`<?xml version="1.0"?><rss version="2.0"><channel>
<item><title>Apple &amp; Nvidia</title><link> https://example.com/a </link><pubDate>Tue, 06 Oct 2026 14:00:00 +0000</pubDate><description><![CDATA[<p>Record <b>quarter</b></p>]]></description></item>
<item><title>GMT dated</title><link>https://example.com/b</link><pubDate>Tue, 06 Oct 2026 13:00:00 GMT</pubDate><description>&lt;a href="x"&gt;escaped&lt;/a&gt;&amp;nbsp;html</description></item>
<item><title>Undated</title><link>https://example.com/c</link><pubDate>yesterday</pubDate></item>
</channel></rss>`)

	rssItems, err := utilities.NewRssFeedParser().Parse(feedBody)

	require.NoError(t, err)
	require.Len(t, rssItems, 2)
	assert.Equal(t, utilities.RssItem{Title: "Apple & Nvidia", Link: "https://example.com/a", PublishedAt: time.Date(2026, 10, 6, 14, 0, 0, 0, time.UTC), Description: "Record quarter"}, utilities.RssItem{Title: rssItems[0].Title, Link: rssItems[0].Link, PublishedAt: rssItems[0].PublishedAt.UTC(), Description: rssItems[0].Description})
	assert.Equal(t, "escaped html", rssItems[1].Description)
	assert.True(t, rssItems[1].PublishedAt.Equal(time.Date(2026, 10, 6, 13, 0, 0, 0, time.UTC)))
}

func TestRssFeedParser_RejectsMalformedFeeds(t *testing.T) {
	_, err := utilities.NewRssFeedParser().Parse([]byte(`<rss><channel><item>`))

	assert.Error(t, err)
}

func TestRssFeedParser_RejectsWellFormedDocumentsThatAreNotRss(t *testing.T) {
	_, err := utilities.NewRssFeedParser().Parse([]byte(`<error><message>rate limited</message></error>`))

	assert.Error(t, err)
}

func TestRssFeedParser_RemovesTheOutletSuffixAddedByAggregators(t *testing.T) {
	feedBody := []byte(`<rss><channel><item><title>台積電法說會 - 鉅亨網</title><link>x</link><pubDate>Tue, 06 Oct 2026 14:00:00 +0000</pubDate><source url="https://news.cnyes.com">鉅亨網</source></item><item><title>Fed holds rates - live</title><link>y</link><pubDate>Tue, 06 Oct 2026 14:00:00 +0000</pubDate></item></channel></rss>`)

	rssItems, err := utilities.NewRssFeedParser().Parse(feedBody)

	require.NoError(t, err)
	assert.Equal(t, "台積電法說會", rssItems[0].Title)
	assert.Equal(t, "Fed holds rates - live", rssItems[1].Title)
}

func TestRssFeedParser_KeepsEscapedAngleBracketsAsText(t *testing.T) {
	feedBody := []byte(`<rss><channel><item><title>Rates</title><link>x</link><pubDate>Tue, 06 Oct 2026 14:00:00 +0000</pubDate><description>S&amp;amp;P &amp;lt; 5000 and yields &amp;gt; 4%</description></item></channel></rss>`)

	rssItems, err := utilities.NewRssFeedParser().Parse(feedBody)

	require.NoError(t, err)
	assert.Equal(t, "S&P < 5000 and yields > 4%", rssItems[0].Description)
}

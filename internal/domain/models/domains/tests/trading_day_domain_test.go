package domains_test

import (
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/stretchr/testify/assert"
)

var taipei = time.FixedZone("Asia/Taipei", 8*60*60)

func taipeiTime(day int, hour int, minute int) time.Time {
	return time.Date(2026, 10, day, hour, minute, 0, 0, taipei)
}

func TestTradingDayDomain_DueSession(t *testing.T) {
	// 2026-10-05 is a Monday, 2026-10-07 a Wednesday, 2026-10-10 a Saturday
	testCases := []struct {
		name               string
		now                time.Time
		expectedSession    string
		expectedIsDue      bool
		expectedNewsSince  time.Time
		expectedNewsBefore time.Time
	}{
		{name: "Wednesday 08:00 runs pre-market with news since Tuesday's close", now: taipeiTime(7, 8, 0), expectedSession: "preMarket", expectedIsDue: true, expectedNewsSince: taipeiTime(6, 13, 30), expectedNewsBefore: taipeiTime(7, 9, 0)},
		{name: "Monday 08:30 runs pre-market with news since Friday's close", now: taipeiTime(5, 8, 30), expectedSession: "preMarket", expectedIsDue: true, expectedNewsSince: taipeiTime(2, 13, 30), expectedNewsBefore: taipeiTime(5, 9, 0)},
		{name: "Wednesday 12:30 runs intraday with news since 09:00", now: taipeiTime(7, 12, 30), expectedSession: "intraday", expectedIsDue: true, expectedNewsSince: taipeiTime(7, 9, 0), expectedNewsBefore: taipeiTime(7, 13, 30)},
		{name: "Wednesday 20:00 runs after-market with news since 13:30", now: taipeiTime(7, 20, 0), expectedSession: "afterMarket", expectedIsDue: true, expectedNewsSince: taipeiTime(7, 13, 30), expectedNewsBefore: taipeiTime(8, 0, 0)},
		{name: "Wednesday 23:59 still runs after-market", now: taipeiTime(7, 23, 59), expectedSession: "afterMarket", expectedIsDue: true, expectedNewsSince: taipeiTime(7, 13, 30), expectedNewsBefore: taipeiTime(8, 0, 0)},
		{name: "Wednesday 09:00 sharp runs nothing", now: taipeiTime(7, 9, 0)},
		{name: "Wednesday 07:59 runs nothing", now: taipeiTime(7, 7, 59)},
		{name: "Wednesday 13:30 sharp runs nothing", now: taipeiTime(7, 13, 30)},
		{name: "Wednesday 14:00 after a missed intraday window runs nothing", now: taipeiTime(7, 14, 0)},
		{name: "Saturday 08:30 runs nothing", now: taipeiTime(10, 8, 30)},
		{name: "a UTC clock is read in Taipei time", now: time.Date(2026, 10, 7, 0, 30, 0, 0, time.UTC), expectedSession: "preMarket", expectedIsDue: true, expectedNewsSince: taipeiTime(6, 13, 30), expectedNewsBefore: taipeiTime(7, 9, 0)},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			tradingDay := domains.NewTradingDayDomain(testCase.now)

			session, isDue := tradingDay.DueSession()

			assert.Equal(t, testCase.expectedIsDue, isDue)
			assert.Equal(t, testCase.expectedSession, session)
			if isDue {
				newsPublishedWindow := tradingDay.NewsPublishedWindow(session)
				assert.True(t, testCase.expectedNewsSince.Equal(newsPublishedWindow.Since), "news since %v", newsPublishedWindow.Since)
				assert.True(t, testCase.expectedNewsBefore.Equal(newsPublishedWindow.Before), "news before %v", newsPublishedWindow.Before)
			}
		})
	}
}

func TestTradingDayDomain_ValueIsTheTaipeiDate(t *testing.T) {
	assert.Equal(t, "2026-10-07", domains.NewTradingDayDomain(time.Date(2026, 10, 6, 23, 0, 0, 0, time.UTC)).Value())
}

func TestTradingDayDomain_WeekendsAreNotTradingDays(t *testing.T) {
	assert.False(t, domains.NewTradingDayDomain(taipeiTime(10, 12, 0)).IsTradingDay())
	assert.False(t, domains.NewTradingDayDomain(taipeiTime(11, 12, 0)).IsTradingDay())
	assert.True(t, domains.NewTradingDayDomain(taipeiTime(9, 12, 0)).IsTradingDay())
}

func TestTradingDayDomain_UnknownSessionHasNoNewsWindow(t *testing.T) {
	assert.Equal(t, vo.NewsPublishedWindowVo{}, domains.NewTradingDayDomain(taipeiTime(7, 8, 0)).NewsPublishedWindow("lunchBreak"))
}

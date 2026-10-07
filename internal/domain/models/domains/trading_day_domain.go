package domains

import (
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

const (
	TradingSessionPreMarket   = "preMarket"
	TradingSessionIntraday    = "intraday"
	TradingSessionAfterMarket = "afterMarket"
	tradingDayLayout          = "2006-01-02"
)

// Taiwan has no daylight saving time, so a fixed offset avoids depending on the host's time zone database
var taipeiTimeZone = time.FixedZone("Asia/Taipei", 8*60*60)

var taiwanStockTradingSessionWindows = []vo.TradingSessionWindowVo{
	{Session: TradingSessionPreMarket, OpensAt: 8 * time.Hour, ClosesAt: 9 * time.Hour, NewsSince: 13*time.Hour + 30*time.Minute, IsNewsSinceOnPreviousTradingDay: true},
	{Session: TradingSessionIntraday, OpensAt: 12*time.Hour + 30*time.Minute, ClosesAt: 13*time.Hour + 30*time.Minute, NewsSince: 9 * time.Hour},
	{Session: TradingSessionAfterMarket, OpensAt: 20 * time.Hour, ClosesAt: 24 * time.Hour, NewsSince: 13*time.Hour + 30*time.Minute},
}

type TradingDayDomain struct {
	now      time.Time
	midnight time.Time
}

func NewTradingDayDomain(now time.Time) TradingDayDomain {
	taipeiNow := now.In(taipeiTimeZone)
	return TradingDayDomain{now: taipeiNow, midnight: time.Date(taipeiNow.Year(), taipeiNow.Month(), taipeiNow.Day(), 0, 0, 0, 0, taipeiTimeZone)}
}

func (tradingDayDomain TradingDayDomain) IsTradingDay() bool {
	weekday := tradingDayDomain.midnight.Weekday()
	return weekday != time.Saturday && weekday != time.Sunday
}

func (tradingDayDomain TradingDayDomain) Value() string {
	return tradingDayDomain.midnight.Format(tradingDayLayout)
}

func (tradingDayDomain TradingDayDomain) DueSession() (string, bool) {
	if !tradingDayDomain.IsTradingDay() {
		return "", false
	}
	sinceMidnight := tradingDayDomain.now.Sub(tradingDayDomain.midnight)
	for _, tradingSessionWindow := range taiwanStockTradingSessionWindows {
		if sinceMidnight >= tradingSessionWindow.OpensAt && sinceMidnight < tradingSessionWindow.ClosesAt {
			return tradingSessionWindow.Session, true
		}
	}
	return "", false
}

func (tradingDayDomain TradingDayDomain) NewsPublishedSince(session string) time.Time {
	for _, tradingSessionWindow := range taiwanStockTradingSessionWindows {
		if tradingSessionWindow.Session != session {
			continue
		}
		newsDay := tradingDayDomain.midnight
		if tradingSessionWindow.IsNewsSinceOnPreviousTradingDay {
			newsDay = newsDay.AddDate(0, 0, -1)
			for !NewTradingDayDomain(newsDay).IsTradingDay() {
				newsDay = newsDay.AddDate(0, 0, -1)
			}
		}
		return newsDay.Add(tradingSessionWindow.NewsSince)
	}
	return time.Time{}
}

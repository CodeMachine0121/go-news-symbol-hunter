package twse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/cache"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/httpfetch"
	"github.com/shopspring/decimal"
)

const (
	TwsePriceSource           = "證交所"
	twseQuoteCurrency         = "TWD"
	dailyClosingCacheDuration = time.Hour
	republicOfChinaYearOffset = 1911
)

var (
	errTwseClosingPriceMissing = errors.New("TWSE has no closing price for this stock")
	taipeiLocation             = time.FixedZone("Asia/Taipei", 8*60*60)
)

type twseDailyClosing struct {
	Date         string `json:"Date"`
	Code         string `json:"Code"`
	ClosingPrice string `json:"ClosingPrice"`
}

type TwsePriceProxy struct {
	httpBodyReader     *httpfetch.HttpBodyReader
	dailyClosingUrl    string
	dailyClosingsCache *cache.RefreshingCache[map[string]twseDailyClosing]
}

func NewTwsePriceProxy(httpBodyReader *httpfetch.HttpBodyReader, clockProxy interfaces.IClockProxy, dailyClosingUrl string) *TwsePriceProxy {
	twsePriceProxy := &TwsePriceProxy{httpBodyReader: httpBodyReader, dailyClosingUrl: dailyClosingUrl}
	twsePriceProxy.dailyClosingsCache = cache.NewRefreshingCache(clockProxy, dailyClosingCacheDuration, twsePriceProxy.downloadDailyClosings)
	return twsePriceProxy
}

func (twsePriceProxy *TwsePriceProxy) FetchPrice(ctx context.Context, symbol string) (vo.PriceQuoteVo, error) {
	dailyClosingsByCode, err := twsePriceProxy.dailyClosingsCache.Get(ctx)
	if err != nil {
		return vo.PriceQuoteVo{}, err
	}
	dailyClosing, found := dailyClosingsByCode[symbol]
	if !found {
		return vo.PriceQuoteVo{}, errTwseClosingPriceMissing
	}
	closingPrice, err := decimal.NewFromString(strings.ReplaceAll(strings.TrimSpace(dailyClosing.ClosingPrice), ",", ""))
	if err != nil {
		return vo.PriceQuoteVo{}, fmt.Errorf("parse closing price %q: %w", dailyClosing.ClosingPrice, err)
	}
	tradingDate, err := dailyClosing.tradingDate()
	if err != nil {
		return vo.PriceQuoteVo{}, err
	}
	return vo.NewPriceQuoteVo(closingPrice, twseQuoteCurrency, tradingDate, TwsePriceSource)
}

// handed to the cache as its download function
func (twsePriceProxy *TwsePriceProxy) downloadDailyClosings(ctx context.Context) (map[string]twseDailyClosing, error) {
	responseBody, err := twsePriceProxy.httpBodyReader.Read(ctx, twsePriceProxy.dailyClosingUrl)
	if err != nil {
		return nil, err
	}
	var dailyClosings []twseDailyClosing
	if err := json.Unmarshal(responseBody, &dailyClosings); err != nil {
		return nil, err
	}
	if len(dailyClosings) == 0 {
		return nil, errTwseClosingPriceMissing
	}
	dailyClosingsByCode := make(map[string]twseDailyClosing, len(dailyClosings))
	for _, dailyClosing := range dailyClosings {
		dailyClosingsByCode[strings.TrimSpace(dailyClosing.Code)] = dailyClosing
	}
	return dailyClosingsByCode, nil
}

// TWSE dates use the Republic of China calendar, e.g. 1151005 is 2026-10-05
func (dailyClosing twseDailyClosing) tradingDate() (time.Time, error) {
	rawDate := strings.TrimSpace(dailyClosing.Date)
	if len(rawDate) < 7 {
		return time.Time{}, fmt.Errorf("unexpected TWSE date %q", dailyClosing.Date)
	}
	republicOfChinaYear, err := strconv.Atoi(rawDate[:len(rawDate)-4])
	if err != nil {
		return time.Time{}, fmt.Errorf("unexpected TWSE date %q: %w", dailyClosing.Date, err)
	}
	return time.ParseInLocation("20060102", strconv.Itoa(republicOfChinaYear+republicOfChinaYearOffset)+rawDate[len(rawDate)-4:], taipeiLocation)
}

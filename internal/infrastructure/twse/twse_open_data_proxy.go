package twse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/cache"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/httpfetch"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/utilities"
	"github.com/shopspring/decimal"
)

const (
	TwsePriceSource            = "證交所"
	twseQuoteCurrency          = "TWD"
	listedCompanyCacheDuration = 24 * time.Hour
	dailyClosingCacheDuration  = time.Hour
)

var (
	errListedCompaniesEmpty    = errors.New("TWSE returned no listed companies")
	errTwseClosingPriceMissing = errors.New("TWSE has no closing price for this stock")
)

type TwseOpenDataUrls struct {
	ListedCompanies string
	DailyClosing    string
}

type twseListedCompany struct {
	StockCode string `json:"公司代號"`
	ShortName string `json:"公司簡稱"`
}

type twseDailyClosing struct {
	Date         string `json:"Date"`
	Code         string `json:"Code"`
	ClosingPrice string `json:"ClosingPrice"`
}

type TwseOpenDataProxy struct {
	httpBodyReader            *httpfetch.HttpBodyReader
	republicOfChinaDateParser *utilities.RepublicOfChinaDateParser
	openDataUrls              TwseOpenDataUrls
	shortNamesCache           *cache.RefreshingCache[map[string]string]
	dailyClosingsCache        *cache.RefreshingCache[map[string]twseDailyClosing]
}

func NewTwseOpenDataProxy(httpBodyReader *httpfetch.HttpBodyReader, clockProxy interfaces.IClockProxy, openDataUrls TwseOpenDataUrls) *TwseOpenDataProxy {
	twseOpenDataProxy := &TwseOpenDataProxy{httpBodyReader: httpBodyReader, republicOfChinaDateParser: utilities.NewRepublicOfChinaDateParser(), openDataUrls: openDataUrls}
	twseOpenDataProxy.shortNamesCache = cache.NewRefreshingCache(clockProxy, listedCompanyCacheDuration, twseOpenDataProxy.downloadShortNames)
	twseOpenDataProxy.dailyClosingsCache = cache.NewRefreshingCache(clockProxy, dailyClosingCacheDuration, twseOpenDataProxy.downloadDailyClosings)
	return twseOpenDataProxy
}

func (twseOpenDataProxy *TwseOpenDataProxy) FindCompanyShortName(ctx context.Context, stockCode string) (string, bool, error) {
	shortNamesByStockCode, err := twseOpenDataProxy.shortNamesCache.Get(ctx)
	if err != nil {
		return "", false, err
	}
	shortName, found := shortNamesByStockCode[stockCode]
	return shortName, found, nil
}

func (twseOpenDataProxy *TwseOpenDataProxy) FetchPrice(ctx context.Context, symbol string) (vo.PriceQuoteVo, error) {
	dailyClosingsByCode, err := twseOpenDataProxy.dailyClosingsCache.Get(ctx)
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
	tradingDate, err := twseOpenDataProxy.republicOfChinaDateParser.ParseTaipeiMidnight(dailyClosing.Date)
	if err != nil {
		return vo.PriceQuoteVo{}, err
	}
	return vo.NewPriceQuoteVo(closingPrice, twseQuoteCurrency, tradingDate, TwsePriceSource)
}

// handed to the cache as its download function
func (twseOpenDataProxy *TwseOpenDataProxy) downloadShortNames(ctx context.Context) (map[string]string, error) {
	responseBody, err := twseOpenDataProxy.httpBodyReader.Read(ctx, twseOpenDataProxy.openDataUrls.ListedCompanies)
	if err != nil {
		return nil, err
	}
	var listedCompanies []twseListedCompany
	if err := json.Unmarshal(responseBody, &listedCompanies); err != nil {
		return nil, err
	}
	shortNamesByStockCode := make(map[string]string, len(listedCompanies))
	for _, listedCompany := range listedCompanies {
		stockCode := strings.TrimSpace(listedCompany.StockCode)
		shortName := strings.TrimSpace(listedCompany.ShortName)
		if stockCode != "" && shortName != "" {
			shortNamesByStockCode[stockCode] = shortName
		}
	}
	if len(shortNamesByStockCode) == 0 {
		return nil, errListedCompaniesEmpty
	}
	return shortNamesByStockCode, nil
}

// handed to the cache as its download function
func (twseOpenDataProxy *TwseOpenDataProxy) downloadDailyClosings(ctx context.Context) (map[string]twseDailyClosing, error) {
	responseBody, err := twseOpenDataProxy.httpBodyReader.Read(ctx, twseOpenDataProxy.openDataUrls.DailyClosing)
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

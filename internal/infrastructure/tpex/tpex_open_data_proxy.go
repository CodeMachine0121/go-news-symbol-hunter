package tpex

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
)

const (
	TpexPriceSource           = "櫃買中心"
	tpexQuoteCurrency         = "TWD"
	dailyClosingCacheDuration = time.Hour
)

var (
	errTpexDailyClosingsEmpty = errors.New("TPEx returned no daily closing quotes")
	errClosingPriceMissing    = errors.New("TPEx has no closing price for this stock")
)

type tpexDailyClosing struct {
	Date                  string `json:"Date"`
	SecuritiesCompanyCode string `json:"SecuritiesCompanyCode"`
	CompanyName           string `json:"CompanyName"`
	Close                 string `json:"Close"`
}

type TpexOpenDataProxy struct {
	httpBodyReader     *httpfetch.HttpBodyReader
	quoteParser        *utilities.TaiwanExchangeQuoteParser
	dailyClosingUrl    string
	dailyClosingsCache *cache.RefreshingCache[map[string]tpexDailyClosing]
}

func NewTpexOpenDataProxy(httpBodyReader *httpfetch.HttpBodyReader, clockProxy interfaces.IClockProxy, dailyClosingUrl string) *TpexOpenDataProxy {
	tpexOpenDataProxy := &TpexOpenDataProxy{httpBodyReader: httpBodyReader, quoteParser: utilities.NewTaiwanExchangeQuoteParser(), dailyClosingUrl: dailyClosingUrl}
	tpexOpenDataProxy.dailyClosingsCache = cache.NewRefreshingCache(clockProxy, dailyClosingCacheDuration, tpexOpenDataProxy.downloadDailyClosings)
	return tpexOpenDataProxy
}

// the whole directory is returned whatever the query, because it is already cached
func (tpexOpenDataProxy *TpexOpenDataProxy) SearchSymbolListings(ctx context.Context, query string) ([]vo.SymbolListingVo, error) {
	dailyClosingsByCode, err := tpexOpenDataProxy.dailyClosingsCache.Get(ctx)
	if err != nil {
		return nil, err
	}
	listings := make([]vo.SymbolListingVo, 0, len(dailyClosingsByCode))
	for code, dailyClosing := range dailyClosingsByCode {
		// TPEx numbers warrants with six digits starting with 7, and warrants are not analysed
		if len(code) == 6 && strings.HasPrefix(code, "7") {
			continue
		}
		listings = append(listings, vo.SymbolListingVo{Symbol: code, Name: strings.TrimSpace(dailyClosing.CompanyName)})
	}
	return listings, nil
}

func (tpexOpenDataProxy *TpexOpenDataProxy) FetchPrice(ctx context.Context, symbol string) (vo.PriceQuoteVo, error) {
	dailyClosingsByCode, err := tpexOpenDataProxy.dailyClosingsCache.Get(ctx)
	if err != nil {
		return vo.PriceQuoteVo{}, err
	}
	dailyClosing, found := dailyClosingsByCode[symbol]
	if !found {
		return vo.PriceQuoteVo{}, errClosingPriceMissing
	}
	closingPrice, err := tpexOpenDataProxy.quoteParser.ParseClosingPrice(dailyClosing.Close)
	if err != nil {
		return vo.PriceQuoteVo{}, fmt.Errorf("%w: %v", errClosingPriceMissing, err)
	}
	tradingDate, err := tpexOpenDataProxy.quoteParser.ParseTradingDate(dailyClosing.Date)
	if err != nil {
		return vo.PriceQuoteVo{}, err
	}
	return vo.NewPriceQuoteVo(closingPrice, tpexQuoteCurrency, tradingDate, TpexPriceSource)
}

// handed to the cache as its download function
func (tpexOpenDataProxy *TpexOpenDataProxy) downloadDailyClosings(ctx context.Context) (map[string]tpexDailyClosing, error) {
	responseBody, err := tpexOpenDataProxy.httpBodyReader.Read(ctx, tpexOpenDataProxy.dailyClosingUrl)
	if err != nil {
		return nil, err
	}
	var dailyClosings []tpexDailyClosing
	if err := json.Unmarshal(responseBody, &dailyClosings); err != nil {
		return nil, err
	}
	dailyClosingsByCode := make(map[string]tpexDailyClosing, len(dailyClosings))
	for _, dailyClosing := range dailyClosings {
		stockCode := strings.TrimSpace(dailyClosing.SecuritiesCompanyCode)
		if stockCode != "" && strings.TrimSpace(dailyClosing.CompanyName) != "" {
			dailyClosingsByCode[stockCode] = dailyClosing
		}
	}
	if len(dailyClosingsByCode) == 0 {
		return nil, errTpexDailyClosingsEmpty
	}
	return dailyClosingsByCode, nil
}

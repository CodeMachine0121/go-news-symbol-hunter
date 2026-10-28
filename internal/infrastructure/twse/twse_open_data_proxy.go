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
)

const (
	TwsePriceSource           = "證交所"
	twseQuoteCurrency         = "TWD"
	dailyClosingCacheDuration = time.Hour
)

var (
	errDailyClosingsEmpty  = errors.New("TWSE returned no daily closing quotes")
	errClosingPriceMissing = errors.New("TWSE has no closing price for this stock")
)

// the daily closing quotes list ETFs as well as companies, so they double as the directory of listed stocks
type twseDailyClosing struct {
	Date         string `json:"Date"`
	Code         string `json:"Code"`
	Name         string `json:"Name"`
	ClosingPrice string `json:"ClosingPrice"`
}

type TwseOpenDataProxy struct {
	httpBodyReader     *httpfetch.HttpBodyReader
	quoteParser        *utilities.TaiwanExchangeQuoteParser
	dailyClosingUrl    string
	dailyClosingsCache *cache.RefreshingCache[map[string]twseDailyClosing]
}

func NewTwseOpenDataProxy(httpBodyReader *httpfetch.HttpBodyReader, clockProxy interfaces.IClockProxy, dailyClosingUrl string) *TwseOpenDataProxy {
	twseOpenDataProxy := &TwseOpenDataProxy{httpBodyReader: httpBodyReader, quoteParser: utilities.NewTaiwanExchangeQuoteParser(), dailyClosingUrl: dailyClosingUrl}
	twseOpenDataProxy.dailyClosingsCache = cache.NewRefreshingCache(clockProxy, dailyClosingCacheDuration, twseOpenDataProxy.downloadDailyClosings)
	return twseOpenDataProxy
}

// the whole directory is returned whatever the query, because it is already cached
func (twseOpenDataProxy *TwseOpenDataProxy) SearchSymbolListings(ctx context.Context, query string) ([]vo.SymbolListingVo, error) {
	dailyClosingsByCode, err := twseOpenDataProxy.dailyClosingsCache.Get(ctx)
	if err != nil {
		return nil, err
	}
	listings := make([]vo.SymbolListingVo, 0, len(dailyClosingsByCode))
	for code, dailyClosing := range dailyClosingsByCode {
		listings = append(listings, vo.SymbolListingVo{Symbol: code, Name: strings.TrimSpace(dailyClosing.Name)})
	}
	return listings, nil
}

func (twseOpenDataProxy *TwseOpenDataProxy) FetchPrice(ctx context.Context, symbol string) (vo.PriceQuoteVo, error) {
	dailyClosingsByCode, err := twseOpenDataProxy.dailyClosingsCache.Get(ctx)
	if err != nil {
		return vo.PriceQuoteVo{}, err
	}
	dailyClosing, found := dailyClosingsByCode[symbol]
	if !found {
		return vo.PriceQuoteVo{}, errClosingPriceMissing
	}
	closingPrice, err := twseOpenDataProxy.quoteParser.ParseClosingPrice(dailyClosing.ClosingPrice)
	if err != nil {
		return vo.PriceQuoteVo{}, fmt.Errorf("%w: %v", errClosingPriceMissing, err)
	}
	tradingDate, err := twseOpenDataProxy.quoteParser.ParseTradingDate(dailyClosing.Date)
	if err != nil {
		return vo.PriceQuoteVo{}, err
	}
	return vo.NewPriceQuoteVo(closingPrice, twseQuoteCurrency, tradingDate, TwsePriceSource)
}

// handed to the cache as its download function
func (twseOpenDataProxy *TwseOpenDataProxy) downloadDailyClosings(ctx context.Context) (map[string]twseDailyClosing, error) {
	responseBody, err := twseOpenDataProxy.httpBodyReader.Read(ctx, twseOpenDataProxy.dailyClosingUrl)
	if err != nil {
		return nil, err
	}
	var dailyClosings []twseDailyClosing
	if err := json.Unmarshal(responseBody, &dailyClosings); err != nil {
		return nil, err
	}
	dailyClosingsByCode := make(map[string]twseDailyClosing, len(dailyClosings))
	for _, dailyClosing := range dailyClosings {
		code := strings.TrimSpace(dailyClosing.Code)
		if code != "" {
			dailyClosingsByCode[code] = dailyClosing
		}
	}
	if len(dailyClosingsByCode) == 0 {
		return nil, errDailyClosingsEmpty
	}
	return dailyClosingsByCode, nil
}

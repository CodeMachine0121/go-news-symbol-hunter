package yahoofinance

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/httpfetch"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/news"
	"github.com/shopspring/decimal"
)

const (
	YahooFinanceSource = "Yahoo 財經"
)

var errYahooFinanceQuoteMissing = errors.New("yahoo finance returned no usable quote")

// Yahoo also lists foreign exchanges and futures, which are not US stocks; OTC markets count as US
var usExchangeCodes = map[string]bool{"NMS": true, "NGM": true, "NCM": true, "NAS": true, "NYQ": true, "NYS": true, "ASE": true, "PCX": true, "BTS": true, "PNK": true, "OQB": true, "OQX": true}

var usStockQuoteTypes = map[string]bool{"EQUITY": true, "ETF": true}

// a share class such as BRK.B is only found by search in Yahoo's BRK-B form
var shareClassSymbolPattern = regexp.MustCompile(`^[A-Za-z]{1,5}\.[A-Za-z]{1,2}$`)

type YahooFinanceUrls struct {
	HeadlineFeed string
	Chart        string
	Search       string
}

type yahooFinanceChartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Currency           string          `json:"currency"`
				RegularMarketPrice decimal.Decimal `json:"regularMarketPrice"`
				RegularMarketTime  int64           `json:"regularMarketTime"`
			} `json:"meta"`
		} `json:"result"`
	} `json:"chart"`
}

type yahooFinanceSearchResponse struct {
	Quotes []struct {
		Symbol    string `json:"symbol"`
		ShortName string `json:"shortname"`
		LongName  string `json:"longname"`
		QuoteType string `json:"quoteType"`
		Exchange  string `json:"exchange"`
	} `json:"quotes"`
}

type yahooFinanceSymbol string

type yahooFinanceTicker string

type YahooFinanceProxy struct {
	httpBodyReader *httpfetch.HttpBodyReader
	rssNewsReader  *news.RssNewsReader
	urls           YahooFinanceUrls
}

func NewYahooFinanceProxy(httpBodyReader *httpfetch.HttpBodyReader, rssNewsReader *news.RssNewsReader, urls YahooFinanceUrls) *YahooFinanceProxy {
	return &YahooFinanceProxy{httpBodyReader: httpBodyReader, rssNewsReader: rssNewsReader, urls: urls}
}

func (yahooFinanceProxy *YahooFinanceProxy) ProviderName() string {
	return YahooFinanceSource
}

func (yahooFinanceProxy *YahooFinanceProxy) FetchNews(ctx context.Context, searchKeyword string) ([]vo.NewsVo, error) {
	query := url.Values{"s": {yahooFinanceSymbol(searchKeyword).toTicker()}, "region": {"US"}, "lang": {"en-US"}}
	return yahooFinanceProxy.rssNewsReader.ReadNews(ctx, yahooFinanceProxy.urls.HeadlineFeed+"?"+query.Encode(), YahooFinanceSource, true)
}

func (yahooFinanceProxy *YahooFinanceProxy) FetchPrice(ctx context.Context, symbol string) (vo.PriceQuoteVo, error) {
	chartResponse, err := yahooFinanceProxy.readChart(ctx, symbol)
	if err != nil {
		return vo.PriceQuoteVo{}, err
	}
	if len(chartResponse.Chart.Result) == 0 || chartResponse.Chart.Result[0].Meta.RegularMarketTime == 0 || strings.TrimSpace(chartResponse.Chart.Result[0].Meta.Currency) == "" {
		return vo.PriceQuoteVo{}, errYahooFinanceQuoteMissing
	}
	meta := chartResponse.Chart.Result[0].Meta
	return vo.NewPriceQuoteVo(meta.RegularMarketPrice, strings.TrimSpace(meta.Currency), time.Unix(meta.RegularMarketTime, 0).UTC(), YahooFinanceSource)
}

func (yahooFinanceProxy *YahooFinanceProxy) SearchSymbolListings(ctx context.Context, query string) ([]vo.SymbolListingVo, error) {
	searchQuery := url.Values{"q": {yahooFinanceSymbol(query).toSearchQuery()}, "quotesCount": {"20"}, "newsCount": {"0"}}
	responseBody, err := yahooFinanceProxy.httpBodyReader.Read(ctx, yahooFinanceProxy.urls.Search+"?"+searchQuery.Encode())
	if err != nil {
		return nil, err
	}
	var searchResponse yahooFinanceSearchResponse
	if err := json.Unmarshal(responseBody, &searchResponse); err != nil {
		return nil, err
	}
	listings := []vo.SymbolListingVo{}
	for _, quote := range searchResponse.Quotes {
		if !usExchangeCodes[quote.Exchange] || !usStockQuoteTypes[quote.QuoteType] {
			continue
		}
		// the short name is cut to a fixed width, so the long name is preferred
		companyName := strings.TrimSpace(quote.LongName)
		if companyName == "" {
			companyName = strings.TrimSpace(quote.ShortName)
		}
		listings = append(listings, vo.SymbolListingVo{Symbol: yahooFinanceTicker(quote.Symbol).toSymbol(), Name: companyName, Priority: len(listings)})
	}
	return listings, nil
}

func (yahooFinanceProxy *YahooFinanceProxy) readChart(ctx context.Context, symbol string) (yahooFinanceChartResponse, error) {
	chartUrl := strings.TrimRight(yahooFinanceProxy.urls.Chart, "/") + "/" + url.PathEscape(yahooFinanceSymbol(symbol).toTicker()) + "?range=1d&interval=1d"
	responseBody, err := yahooFinanceProxy.httpBodyReader.Read(ctx, chartUrl)
	if err != nil {
		return yahooFinanceChartResponse{}, err
	}
	var chartResponse yahooFinanceChartResponse
	if err := json.Unmarshal(responseBody, &chartResponse); err != nil {
		return yahooFinanceChartResponse{}, err
	}
	return chartResponse, nil
}

// Yahoo writes share classes with a dash: BRK.B is BRK-B
func (symbol yahooFinanceSymbol) toTicker() string {
	return strings.ReplaceAll(string(symbol), ".", "-")
}

func (symbol yahooFinanceSymbol) toSearchQuery() string {
	if shareClassSymbolPattern.MatchString(string(symbol)) {
		return symbol.toTicker()
	}
	return string(symbol)
}

// symbols are kept the way users write share classes: BRK-B is BRK.B
func (ticker yahooFinanceTicker) toSymbol() string {
	return strings.ReplaceAll(string(ticker), "-", ".")
}

package yahoofinance

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
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

type YahooFinanceUrls struct {
	HeadlineFeed string
	Chart        string
}

type yahooFinanceChartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Currency           string          `json:"currency"`
				RegularMarketPrice decimal.Decimal `json:"regularMarketPrice"`
				RegularMarketTime  int64           `json:"regularMarketTime"`
				ShortName          string          `json:"shortName"`
				LongName           string          `json:"longName"`
			} `json:"meta"`
		} `json:"result"`
	} `json:"chart"`
}

type yahooFinanceSymbol string

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

func (yahooFinanceProxy *YahooFinanceProxy) FindCompanyShortName(ctx context.Context, symbol string) (string, bool, error) {
	chartResponse, err := yahooFinanceProxy.readChart(ctx, symbol)
	if err != nil {
		return "", false, err
	}
	if len(chartResponse.Chart.Result) == 0 {
		return "", false, nil
	}
	meta := chartResponse.Chart.Result[0].Meta
	companyName := strings.TrimSpace(meta.ShortName)
	if companyName == "" {
		companyName = strings.TrimSpace(meta.LongName)
	}
	return companyName, companyName != "", nil
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

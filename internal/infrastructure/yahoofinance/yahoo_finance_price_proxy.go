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
	"github.com/shopspring/decimal"
)

const YahooFinancePriceSource = "Yahoo 財經"

var errYahooFinanceQuoteMissing = errors.New("yahoo finance returned no quote")

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

type YahooFinancePriceProxy struct {
	httpBodyReader *httpfetch.HttpBodyReader
	chartUrl       string
}

func NewYahooFinancePriceProxy(httpBodyReader *httpfetch.HttpBodyReader, chartUrl string) *YahooFinancePriceProxy {
	return &YahooFinancePriceProxy{httpBodyReader: httpBodyReader, chartUrl: chartUrl}
}

func (yahooFinancePriceProxy *YahooFinancePriceProxy) FetchPrice(ctx context.Context, symbol string) (vo.PriceQuoteVo, error) {
	responseBody, err := yahooFinancePriceProxy.httpBodyReader.Read(ctx, strings.TrimRight(yahooFinancePriceProxy.chartUrl, "/")+"/"+url.PathEscape(symbol)+"?range=1d&interval=1d")
	if err != nil {
		return vo.PriceQuoteVo{}, err
	}
	var chartResponse yahooFinanceChartResponse
	if err := json.Unmarshal(responseBody, &chartResponse); err != nil {
		return vo.PriceQuoteVo{}, err
	}
	if len(chartResponse.Chart.Result) == 0 || chartResponse.Chart.Result[0].Meta.RegularMarketTime == 0 {
		return vo.PriceQuoteVo{}, errYahooFinanceQuoteMissing
	}
	meta := chartResponse.Chart.Result[0].Meta
	return vo.NewPriceQuoteVo(meta.RegularMarketPrice, meta.Currency, time.Unix(meta.RegularMarketTime, 0).UTC(), YahooFinancePriceSource)
}

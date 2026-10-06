package binance

import (
	"context"
	"encoding/json"
	"net/url"

	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/httpfetch"
	"github.com/shopspring/decimal"
)

const (
	BinancePriceSource   = "Binance"
	binanceQuoteCurrency = "USDT"
)

type binanceTickerPrice struct {
	Price decimal.Decimal `json:"price"`
}

type BinancePriceProxy struct {
	httpBodyReader *httpfetch.HttpBodyReader
	clockProxy     interfaces.IClockProxy
	tickerPriceUrl string
}

func NewBinancePriceProxy(httpBodyReader *httpfetch.HttpBodyReader, clockProxy interfaces.IClockProxy, tickerPriceUrl string) *BinancePriceProxy {
	return &BinancePriceProxy{httpBodyReader: httpBodyReader, clockProxy: clockProxy, tickerPriceUrl: tickerPriceUrl}
}

func (binancePriceProxy *BinancePriceProxy) FetchPrice(ctx context.Context, symbol string) (vo.PriceQuoteVo, error) {
	responseBody, err := binancePriceProxy.httpBodyReader.Read(ctx, binancePriceProxy.tickerPriceUrl+"?symbol="+url.QueryEscape(symbol+binanceQuoteCurrency))
	if err != nil {
		return vo.PriceQuoteVo{}, err
	}
	var tickerPrice binanceTickerPrice
	if err := json.Unmarshal(responseBody, &tickerPrice); err != nil {
		return vo.PriceQuoteVo{}, err
	}
	return vo.NewPriceQuoteVo(tickerPrice.Price, binanceQuoteCurrency, binancePriceProxy.clockProxy.Now(), BinancePriceSource)
}

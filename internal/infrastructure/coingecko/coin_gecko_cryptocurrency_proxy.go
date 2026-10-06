package coingecko

import (
	"encoding/json"
	"net/url"
	"strings"
	"sync"
	"time"

	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/utilities"
)

const coinLookupCacheDuration = 24 * time.Hour

type coinGeckoSearchResponse struct {
	Coins []coinGeckoCoin `json:"coins"`
}

type coinGeckoCoin struct {
	Symbol        string `json:"symbol"`
	Name          string `json:"name"`
	MarketCapRank *int   `json:"market_cap_rank"`
}

type cachedCoinLookup struct {
	coinName  string
	found     bool
	expiresAt time.Time
}

type CoinGeckoCryptocurrencyProxy struct {
	httpBodyReader      *utilities.HttpBodyReader
	clockProxy          interfaces.IClockProxy
	searchUrl           string
	cacheMutex          sync.RWMutex
	coinLookupsBySymbol map[string]cachedCoinLookup
}

func NewCoinGeckoCryptocurrencyProxy(httpBodyReader *utilities.HttpBodyReader, clockProxy interfaces.IClockProxy, searchUrl string) *CoinGeckoCryptocurrencyProxy {
	return &CoinGeckoCryptocurrencyProxy{httpBodyReader: httpBodyReader, clockProxy: clockProxy, searchUrl: searchUrl, coinLookupsBySymbol: map[string]cachedCoinLookup{}}
}

func (coinGeckoCryptocurrencyProxy *CoinGeckoCryptocurrencyProxy) FindCoinName(symbol string) (string, bool, error) {
	now := coinGeckoCryptocurrencyProxy.clockProxy.Now()
	if cachedLookup, cached := coinGeckoCryptocurrencyProxy.cachedCoinLookup(symbol); cached && now.Before(cachedLookup.expiresAt) {
		return cachedLookup.coinName, cachedLookup.found, nil
	}
	// fetched without holding the lock so lookups of other symbols never queue behind a slow request
	responseBody, err := coinGeckoCryptocurrencyProxy.httpBodyReader.Read(coinGeckoCryptocurrencyProxy.searchUrl + "?query=" + url.QueryEscape(symbol))
	if err != nil {
		return "", false, err
	}
	var searchResponse coinGeckoSearchResponse
	if err := json.Unmarshal(responseBody, &searchResponse); err != nil {
		return "", false, err
	}
	bestMatch := cachedCoinLookup{expiresAt: now.Add(coinLookupCacheDuration)}
	bestMarketCapRank := 0
	for _, coin := range searchResponse.Coins {
		if !strings.EqualFold(coin.Symbol, symbol) {
			continue
		}
		hasBetterRank := coin.MarketCapRank != nil && (bestMarketCapRank == 0 || *coin.MarketCapRank < bestMarketCapRank)
		if !bestMatch.found || hasBetterRank {
			bestMatch.coinName = coin.Name
			bestMatch.found = true
			if coin.MarketCapRank != nil {
				bestMarketCapRank = *coin.MarketCapRank
			}
		}
	}
	coinGeckoCryptocurrencyProxy.cacheMutex.Lock()
	coinGeckoCryptocurrencyProxy.coinLookupsBySymbol[symbol] = bestMatch
	coinGeckoCryptocurrencyProxy.cacheMutex.Unlock()
	return bestMatch.coinName, bestMatch.found, nil
}

// scopes the read lock so it is released before any network call
func (coinGeckoCryptocurrencyProxy *CoinGeckoCryptocurrencyProxy) cachedCoinLookup(symbol string) (cachedCoinLookup, bool) {
	coinGeckoCryptocurrencyProxy.cacheMutex.RLock()
	defer coinGeckoCryptocurrencyProxy.cacheMutex.RUnlock()
	cachedLookup, cached := coinGeckoCryptocurrencyProxy.coinLookupsBySymbol[symbol]
	return cachedLookup, cached
}

package coingecko

import (
	"context"
	"encoding/json"
	"math"
	"net/url"
	"strings"
	"sync"
	"time"

	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/httpfetch"
)

const (
	coinLookupCacheDuration  = 24 * time.Hour
	maximumCachedCoinLookups = 1000
)

type coinGeckoSearchResponse struct {
	Coins []coinGeckoCoin `json:"coins"`
}

type coinGeckoCoin struct {
	Symbol        string `json:"symbol"`
	Name          string `json:"name"`
	MarketCapRank *int   `json:"market_cap_rank"`
}

type cachedCoinListings struct {
	listings  []vo.SymbolListingVo
	expiresAt time.Time
}

type CoinGeckoCryptocurrencyProxy struct {
	httpBodyReader      *httpfetch.HttpBodyReader
	clockProxy          interfaces.IClockProxy
	searchUrl           string
	cacheMutex          sync.RWMutex
	coinListingsByQuery map[string]cachedCoinListings
}

func NewCoinGeckoCryptocurrencyProxy(httpBodyReader *httpfetch.HttpBodyReader, clockProxy interfaces.IClockProxy, searchUrl string) *CoinGeckoCryptocurrencyProxy {
	return &CoinGeckoCryptocurrencyProxy{httpBodyReader: httpBodyReader, clockProxy: clockProxy, searchUrl: searchUrl, coinListingsByQuery: map[string]cachedCoinListings{}}
}

func (coinGeckoCryptocurrencyProxy *CoinGeckoCryptocurrencyProxy) SearchSymbolListings(ctx context.Context, query string) ([]vo.SymbolListingVo, error) {
	trimmedQuery := strings.TrimSpace(query)
	cacheKey := strings.ToLower(trimmedQuery)
	now := coinGeckoCryptocurrencyProxy.clockProxy.Now()
	if cachedListings, cached := coinGeckoCryptocurrencyProxy.cachedCoinListings(cacheKey); cached && now.Before(cachedListings.expiresAt) {
		return cachedListings.listings, nil
	}
	// fetched without holding the lock so searches for other queries never queue behind a slow request
	responseBody, err := coinGeckoCryptocurrencyProxy.httpBodyReader.Read(ctx, coinGeckoCryptocurrencyProxy.searchUrl+"?query="+url.QueryEscape(trimmedQuery))
	if err != nil {
		return nil, err
	}
	var searchResponse coinGeckoSearchResponse
	if err := json.Unmarshal(responseBody, &searchResponse); err != nil {
		return nil, err
	}
	listings := make([]vo.SymbolListingVo, 0, len(searchResponse.Coins))
	for _, coin := range searchResponse.Coins {
		// an unranked coin is listed after every ranked one
		priority := math.MaxInt
		if coin.MarketCapRank != nil {
			priority = *coin.MarketCapRank
		}
		listings = append(listings, vo.SymbolListingVo{Symbol: coin.Symbol, Name: coin.Name, Priority: priority})
	}
	coinGeckoCryptocurrencyProxy.cacheCoinListings(cacheKey, cachedCoinListings{listings: listings, expiresAt: now.Add(coinLookupCacheDuration)}, now)
	return listings, nil
}

// scopes the read lock so it is released before any network call
func (coinGeckoCryptocurrencyProxy *CoinGeckoCryptocurrencyProxy) cachedCoinListings(cacheKey string) (cachedCoinListings, bool) {
	coinGeckoCryptocurrencyProxy.cacheMutex.RLock()
	defer coinGeckoCryptocurrencyProxy.cacheMutex.RUnlock()
	cachedListings, cached := coinGeckoCryptocurrencyProxy.coinListingsByQuery[cacheKey]
	return cachedListings, cached
}

// scopes the write lock; the cache stays bounded because arbitrary queries come from user input
func (coinGeckoCryptocurrencyProxy *CoinGeckoCryptocurrencyProxy) cacheCoinListings(cacheKey string, coinListings cachedCoinListings, now time.Time) {
	coinGeckoCryptocurrencyProxy.cacheMutex.Lock()
	defer coinGeckoCryptocurrencyProxy.cacheMutex.Unlock()
	if len(coinGeckoCryptocurrencyProxy.coinListingsByQuery) >= maximumCachedCoinLookups {
		for cachedQuery, cachedListings := range coinGeckoCryptocurrencyProxy.coinListingsByQuery {
			if !now.Before(cachedListings.expiresAt) {
				delete(coinGeckoCryptocurrencyProxy.coinListingsByQuery, cachedQuery)
			}
		}
	}
	if len(coinGeckoCryptocurrencyProxy.coinListingsByQuery) < maximumCachedCoinLookups {
		coinGeckoCryptocurrencyProxy.coinListingsByQuery[cacheKey] = coinListings
	}
}

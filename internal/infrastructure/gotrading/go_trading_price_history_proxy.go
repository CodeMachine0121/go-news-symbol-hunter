package gotrading

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

const (
	maximumResponseBodyBytes = 10 << 20
	cryptoQuoteCurrency      = "USDT"
)

var errGoTradingAccountMissing = errors.New("go-trading account is not configured")

type GoTradingAccount struct {
	BaseUrl  string
	Email    string
	Password string
}

// signs in with the account and keeps the access token in memory only; a rejected token is replaced by
// signing in again, so no single-use refresh token ever has to be stored
type GoTradingPriceHistoryProxy struct {
	httpClient       *http.Client
	account          GoTradingAccount
	accessTokenMutex sync.Mutex
	accessToken      string
}

func NewGoTradingPriceHistoryProxy(httpClient *http.Client, account GoTradingAccount) *GoTradingPriceHistoryProxy {
	return &GoTradingPriceHistoryProxy{httpClient: httpClient, account: account}
}

func (goTradingPriceHistoryProxy *GoTradingPriceHistoryProxy) FetchCandles(ctx context.Context, priceHistoryQuery vo.PriceHistoryQueryVo) ([]vo.PriceCandleVo, error) {
	account := goTradingPriceHistoryProxy.account
	if account.BaseUrl == "" || account.Email == "" || account.Password == "" {
		return nil, errGoTradingAccountMissing
	}
	symbol := priceHistoryQuery.Symbol
	if priceHistoryQuery.Category == vo.MarketCategoryCrypto {
		symbol += cryptoQuoteCurrency
	}
	seriesUrl := account.BaseUrl + "/k-candles/series?" + url.Values{
		"symbol":    {symbol},
		"interval":  {priceHistoryQuery.Interval},
		"startTime": {priceHistoryQuery.Since.UTC().Format(time.RFC3339)},
		"endTime":   {priceHistoryQuery.Until.UTC().Format(time.RFC3339)},
	}.Encode()
	rejectedAccessToken := ""
	// a token that expired between requests is rejected once; signing in again and retrying covers it
	for range 2 {
		accessToken, err := goTradingPriceHistoryProxy.accessTokenReplacing(ctx, rejectedAccessToken)
		if err != nil {
			return nil, err
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, seriesUrl, nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Authorization", "Bearer "+accessToken)
		response, err := goTradingPriceHistoryProxy.httpClient.Do(request)
		if err != nil {
			return nil, err
		}
		if response.StatusCode == http.StatusUnauthorized {
			response.Body.Close()
			rejectedAccessToken = accessToken
			continue
		}
		var kCandleSeries goTradingKCandleSeries
		decodeError := json.NewDecoder(io.LimitReader(response.Body, maximumResponseBodyBytes)).Decode(&kCandleSeries)
		response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("go-trading answered %d for %s candles of %s", response.StatusCode, priceHistoryQuery.Interval, symbol)
		}
		if decodeError != nil {
			return nil, decodeError
		}
		priceCandles := make([]vo.PriceCandleVo, 0, len(kCandleSeries.KCandles))
		for _, kCandle := range kCandleSeries.KCandles {
			priceCandles = append(priceCandles, vo.PriceCandleVo{OpenTime: kCandle.OpenTime, Close: kCandle.Close})
		}
		return priceCandles, nil
	}
	return nil, errors.New("go-trading rejected a freshly issued access token")
}

// kept as its own method so the deferred unlock bounds the sign-in: concurrent fetches wait for one sign-in
// instead of each signing in, and the lock is released before the candle request goes out
func (goTradingPriceHistoryProxy *GoTradingPriceHistoryProxy) accessTokenReplacing(ctx context.Context, rejectedAccessToken string) (string, error) {
	goTradingPriceHistoryProxy.accessTokenMutex.Lock()
	defer goTradingPriceHistoryProxy.accessTokenMutex.Unlock()
	if goTradingPriceHistoryProxy.accessToken != "" && goTradingPriceHistoryProxy.accessToken != rejectedAccessToken {
		return goTradingPriceHistoryProxy.accessToken, nil
	}
	account := goTradingPriceHistoryProxy.account
	// marshalling two strings cannot fail
	requestBody, _ := json.Marshal(goTradingSessionRequest{Email: account.Email, Password: account.Password})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, account.BaseUrl+"/sessions", bytes.NewReader(requestBody))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := goTradingPriceHistoryProxy.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	var sessionTokens goTradingSessionTokens
	// a refused sign-in carries no access token, whatever else its body says
	if err := json.NewDecoder(io.LimitReader(response.Body, maximumResponseBodyBytes)).Decode(&sessionTokens); err != nil || sessionTokens.AccessToken == "" {
		return "", fmt.Errorf("go-trading issued no access token (status %d)", response.StatusCode)
	}
	goTradingPriceHistoryProxy.accessToken = sessionTokens.AccessToken
	return sessionTokens.AccessToken, nil
}

package twse

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/httpfetch"
	"golang.org/x/sync/singleflight"
)

const listedCompanyCacheDuration = 24 * time.Hour

var errListedCompaniesEmpty = errors.New("TWSE returned no listed companies")

type twseListedCompany struct {
	StockCode string `json:"公司代號"`
	ShortName string `json:"公司簡稱"`
}

type TwseListedCompanyProxy struct {
	httpBodyReader        *httpfetch.HttpBodyReader
	clockProxy            interfaces.IClockProxy
	listedCompaniesUrl    string
	refreshGroup          singleflight.Group
	cacheMutex            sync.RWMutex
	shortNamesByStockCode map[string]string
	cacheExpiresAt        time.Time
}

func NewTwseListedCompanyProxy(httpBodyReader *httpfetch.HttpBodyReader, clockProxy interfaces.IClockProxy, listedCompaniesUrl string) *TwseListedCompanyProxy {
	return &TwseListedCompanyProxy{httpBodyReader: httpBodyReader, clockProxy: clockProxy, listedCompaniesUrl: listedCompaniesUrl}
}

func (twseListedCompanyProxy *TwseListedCompanyProxy) FindCompanyShortName(ctx context.Context, stockCode string) (string, bool, error) {
	now := twseListedCompanyProxy.clockProxy.Now()
	shortNamesByStockCode, isFresh := twseListedCompanyProxy.cachedShortNames(now)
	if !isFresh {
		// one download serves every concurrent caller; it outlives a single caller's cancellation
		refreshedShortNames, err, _ := twseListedCompanyProxy.refreshGroup.Do("listedCompanies", func() (any, error) {
			return twseListedCompanyProxy.downloadShortNames(context.WithoutCancel(ctx), now)
		})
		if err != nil {
			return "", false, err
		}
		shortNamesByStockCode = refreshedShortNames.(map[string]string)
	}
	shortName, found := shortNamesByStockCode[stockCode]
	return shortName, found, nil
}

// scopes the read lock so it is released before any network call
func (twseListedCompanyProxy *TwseListedCompanyProxy) cachedShortNames(now time.Time) (map[string]string, bool) {
	twseListedCompanyProxy.cacheMutex.RLock()
	defer twseListedCompanyProxy.cacheMutex.RUnlock()
	return twseListedCompanyProxy.shortNamesByStockCode, twseListedCompanyProxy.shortNamesByStockCode != nil && now.Before(twseListedCompanyProxy.cacheExpiresAt)
}

// runs inside singleflight, which only accepts a function value
func (twseListedCompanyProxy *TwseListedCompanyProxy) downloadShortNames(ctx context.Context, now time.Time) (map[string]string, error) {
	responseBody, err := twseListedCompanyProxy.httpBodyReader.Read(ctx, twseListedCompanyProxy.listedCompaniesUrl)
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
	twseListedCompanyProxy.cacheMutex.Lock()
	defer twseListedCompanyProxy.cacheMutex.Unlock()
	twseListedCompanyProxy.shortNamesByStockCode = shortNamesByStockCode
	twseListedCompanyProxy.cacheExpiresAt = now.Add(listedCompanyCacheDuration)
	return shortNamesByStockCode, nil
}

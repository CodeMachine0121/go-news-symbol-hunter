package twse

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/cache"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/httpfetch"
)

const listedCompanyCacheDuration = 24 * time.Hour

var errListedCompaniesEmpty = errors.New("TWSE returned no listed companies")

type twseListedCompany struct {
	StockCode string `json:"公司代號"`
	ShortName string `json:"公司簡稱"`
}

type TwseListedCompanyProxy struct {
	httpBodyReader     *httpfetch.HttpBodyReader
	listedCompaniesUrl string
	shortNamesCache    *cache.RefreshingCache[map[string]string]
}

func NewTwseListedCompanyProxy(httpBodyReader *httpfetch.HttpBodyReader, clockProxy interfaces.IClockProxy, listedCompaniesUrl string) *TwseListedCompanyProxy {
	twseListedCompanyProxy := &TwseListedCompanyProxy{httpBodyReader: httpBodyReader, listedCompaniesUrl: listedCompaniesUrl}
	twseListedCompanyProxy.shortNamesCache = cache.NewRefreshingCache(clockProxy, listedCompanyCacheDuration, twseListedCompanyProxy.downloadShortNames)
	return twseListedCompanyProxy
}

func (twseListedCompanyProxy *TwseListedCompanyProxy) FindCompanyShortName(ctx context.Context, stockCode string) (string, bool, error) {
	shortNamesByStockCode, err := twseListedCompanyProxy.shortNamesCache.Get(ctx)
	if err != nil {
		return "", false, err
	}
	shortName, found := shortNamesByStockCode[stockCode]
	return shortName, found, nil
}

// handed to the cache as its download function
func (twseListedCompanyProxy *TwseListedCompanyProxy) downloadShortNames(ctx context.Context) (map[string]string, error) {
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
	return shortNamesByStockCode, nil
}

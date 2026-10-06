package twse

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/utilities"
)

const listedCompanyCacheDuration = 24 * time.Hour

type twseListedCompany struct {
	StockCode string `json:"公司代號"`
	ShortName string `json:"公司簡稱"`
}

type TwseListedCompanyProxy struct {
	httpBodyReader        *utilities.HttpBodyReader
	clockProxy            interfaces.IClockProxy
	listedCompaniesUrl    string
	cacheMutex            sync.Mutex
	shortNamesByStockCode map[string]string
	cacheExpiresAt        time.Time
}

func NewTwseListedCompanyProxy(httpBodyReader *utilities.HttpBodyReader, clockProxy interfaces.IClockProxy, listedCompaniesUrl string) *TwseListedCompanyProxy {
	return &TwseListedCompanyProxy{httpBodyReader: httpBodyReader, clockProxy: clockProxy, listedCompaniesUrl: listedCompaniesUrl}
}

func (twseListedCompanyProxy *TwseListedCompanyProxy) FindCompanyShortName(stockCode string) (string, bool, error) {
	twseListedCompanyProxy.cacheMutex.Lock()
	defer twseListedCompanyProxy.cacheMutex.Unlock()
	now := twseListedCompanyProxy.clockProxy.Now()
	if twseListedCompanyProxy.shortNamesByStockCode == nil || !now.Before(twseListedCompanyProxy.cacheExpiresAt) {
		responseBody, err := twseListedCompanyProxy.httpBodyReader.Read(twseListedCompanyProxy.listedCompaniesUrl)
		if err != nil {
			return "", false, err
		}
		var listedCompanies []twseListedCompany
		if err := json.Unmarshal(responseBody, &listedCompanies); err != nil {
			return "", false, err
		}
		shortNamesByStockCode := make(map[string]string, len(listedCompanies))
		for _, listedCompany := range listedCompanies {
			shortNamesByStockCode[strings.TrimSpace(listedCompany.StockCode)] = strings.TrimSpace(listedCompany.ShortName)
		}
		twseListedCompanyProxy.shortNamesByStockCode = shortNamesByStockCode
		twseListedCompanyProxy.cacheExpiresAt = now.Add(listedCompanyCacheDuration)
	}
	shortName, found := twseListedCompanyProxy.shortNamesByStockCode[stockCode]
	return shortName, found, nil
}

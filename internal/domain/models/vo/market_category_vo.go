package vo

import "errors"

const (
	MarketCategoryCrypto  = "crypto"
	MarketCategoryTwStock = "twStock"
	MarketCategoryUsStock = "usStock"
)

var ErrMarketCategoryUnsupported = errors.New("市場類別只能是 crypto、twStock、usStock")

type MarketCategoryVo struct {
	Value string
}

func NewMarketCategoryVo(rawCategory string) (MarketCategoryVo, error) {
	switch rawCategory {
	case MarketCategoryCrypto, MarketCategoryTwStock, MarketCategoryUsStock:
		return MarketCategoryVo{Value: rawCategory}, nil
	default:
		return MarketCategoryVo{}, ErrMarketCategoryUnsupported
	}
}

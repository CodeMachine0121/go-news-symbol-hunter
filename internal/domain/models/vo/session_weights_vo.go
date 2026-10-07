package vo

const (
	DefaultPreMarketSessionWeight   = 0.3
	DefaultIntradaySessionWeight    = 0.2
	DefaultAfterMarketSessionWeight = 0.5
)

type SessionWeightsVo struct {
	PreMarket   float64
	Intraday    float64
	AfterMarket float64
}

func NewSessionWeightsVo(preMarket float64, intraday float64, afterMarket float64) SessionWeightsVo {
	sessionWeights := SessionWeightsVo{PreMarket: preMarket, Intraday: intraday, AfterMarket: afterMarket}
	if sessionWeights.PreMarket <= 0 {
		sessionWeights.PreMarket = DefaultPreMarketSessionWeight
	}
	if sessionWeights.Intraday <= 0 {
		sessionWeights.Intraday = DefaultIntradaySessionWeight
	}
	if sessionWeights.AfterMarket <= 0 {
		sessionWeights.AfterMarket = DefaultAfterMarketSessionWeight
	}
	return sessionWeights
}

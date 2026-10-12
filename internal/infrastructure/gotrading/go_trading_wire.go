package gotrading

import (
	"time"

	"github.com/shopspring/decimal"
)

type goTradingSessionRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type goTradingSessionTokens struct {
	AccessToken string `json:"accessToken"`
}

type goTradingKCandleSeries struct {
	KCandles []goTradingKCandle `json:"kCandles"`
}

type goTradingKCandle struct {
	OpenTime time.Time       `json:"openTime"`
	Close    decimal.Decimal `json:"close"`
}

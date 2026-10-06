package interfaces

import "context"

type ICryptocurrencyProxy interface {
	FindCoinName(ctx context.Context, symbol string) (string, bool, error)
}

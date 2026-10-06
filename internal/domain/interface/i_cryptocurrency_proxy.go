package interfaces

type ICryptocurrencyProxy interface {
	FindCoinName(symbol string) (string, bool, error)
}

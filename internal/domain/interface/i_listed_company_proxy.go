package interfaces

type IListedCompanyProxy interface {
	FindCompanyShortName(stockCode string) (string, bool, error)
}

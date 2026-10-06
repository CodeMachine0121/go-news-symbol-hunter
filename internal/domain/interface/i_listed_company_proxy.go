package interfaces

import "context"

type IListedCompanyProxy interface {
	FindCompanyShortName(ctx context.Context, stockCode string) (string, bool, error)
}

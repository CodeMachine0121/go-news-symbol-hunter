package interfaces

import (
	"context"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

// a directory may return more than what matches the query; deciding the match is the domain's job
type ISymbolDirectoryProxy interface {
	SearchSymbolListings(ctx context.Context, query string) ([]vo.SymbolListingVo, error)
}

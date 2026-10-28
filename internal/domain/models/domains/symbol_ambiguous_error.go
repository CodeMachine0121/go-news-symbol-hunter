package domains

import (
	"errors"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
)

var (
	ErrSymbolNotFound  = errors.New("找不到此標的")
	ErrSymbolAmbiguous = errors.New("符合多個標的，請改填代號或更完整的名稱")
)

type SymbolAmbiguousError struct {
	Candidates []dto.SymbolCandidateDto
}

func (symbolAmbiguousError *SymbolAmbiguousError) Error() string {
	return ErrSymbolAmbiguous.Error()
}

func (symbolAmbiguousError *SymbolAmbiguousError) Unwrap() error {
	return ErrSymbolAmbiguous
}

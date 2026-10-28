package service

import (
	"errors"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
)

var ErrSymbolAmbiguous = errors.New("符合多個標的，請改填代號或更完整的名稱")

// carries the candidates so the caller can offer them; matches ErrSymbolAmbiguous through errors.Is
type SymbolAmbiguousError struct {
	Candidates []dto.SymbolCandidateDto
}

func (symbolAmbiguousError *SymbolAmbiguousError) Error() string {
	return ErrSymbolAmbiguous.Error()
}

func (symbolAmbiguousError *SymbolAmbiguousError) Unwrap() error {
	return ErrSymbolAmbiguous
}

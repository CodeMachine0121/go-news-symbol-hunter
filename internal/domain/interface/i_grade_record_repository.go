package interfaces

import (
	"context"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

type IGradeRecordRepository interface {
	// always adds a record; an existing one is never replaced
	Create(ctx context.Context, gradeRecord *entities.GradeRecord) error
	// nil when the analysis event has no grade record
	FindByAnalysisEventID(ctx context.Context, analysisEventID uint) (*entities.GradeRecord, error)
	// each symbol's newest background record on its newest trading day; symbols without one are left out
	FindLatestBackground(ctx context.Context, symbols []vo.SymbolVo) ([]entities.GradeRecord, error)
	// oldest first, at most limit records; manual records only when their analysis event succeeded
	FindForBacktest(ctx context.Context, gradeBacktestFilter vo.GradeBacktestFilterVo, limit int) ([]entities.GradeRecord, error)
}

package interfaces

import (
	"context"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

type IAnalysisResultRepository interface {
	Create(ctx context.Context, analysisResult *entities.AnalysisResult) error
	FindByAnalysisEventID(ctx context.Context, analysisEventID uint) (*entities.AnalysisResult, error)
	// oldest first, at most limit results
	FindForBacktest(ctx context.Context, gradeBacktestFilter vo.GradeBacktestFilterVo, limit int) ([]entities.AnalysisResult, error)
}

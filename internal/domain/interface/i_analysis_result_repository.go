package interfaces

import (
	"context"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
)

type IAnalysisResultRepository interface {
	Create(ctx context.Context, analysisResult *entities.AnalysisResult) error
	FindByAnalysisEventID(ctx context.Context, analysisEventID uint) (*entities.AnalysisResult, error)
}

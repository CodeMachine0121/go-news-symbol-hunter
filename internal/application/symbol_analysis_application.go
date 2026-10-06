package application

import (
	"context"
	"log"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/service"
)

type SymbolAnalysisApplication struct {
	symbolAnalysisService *service.SymbolAnalysisService
}

func NewSymbolAnalysisApplication(symbolAnalysisService *service.SymbolAnalysisService) *SymbolAnalysisApplication {
	return &SymbolAnalysisApplication{symbolAnalysisService: symbolAnalysisService}
}

func (symbolAnalysisApplication *SymbolAnalysisApplication) StartSymbolAnalysis(ctx context.Context, startSymbolAnalysisDto dto.StartSymbolAnalysisDto) (dto.StartedSymbolAnalysisDto, error) {
	startedSymbolAnalysis, err := symbolAnalysisApplication.symbolAnalysisService.StartSymbolAnalysis(ctx, startSymbolAnalysisDto)
	if err != nil || !startedSymbolAnalysis.IsNew {
		return startedSymbolAnalysis, err
	}
	analyzeSymbolDto := dto.AnalyzeSymbolDto{AnalysisEventID: startedSymbolAnalysis.AnalysisEvent.AnalysisEventID, SearchKeyword: startedSymbolAnalysis.SearchKeyword}
	// the analysis outlives the HTTP request that started it
	go symbolAnalysisApplication.AnalyzeSymbol(context.WithoutCancel(ctx), analyzeSymbolDto)
	return startedSymbolAnalysis, nil
}

func (symbolAnalysisApplication *SymbolAnalysisApplication) AnalyzeSymbol(ctx context.Context, analyzeSymbolDto dto.AnalyzeSymbolDto) {
	if err := symbolAnalysisApplication.symbolAnalysisService.AnalyzeSymbol(ctx, analyzeSymbolDto); err != nil {
		log.Printf("analysis event %d could not be recorded: %v", analyzeSymbolDto.AnalysisEventID, err)
	}
}

func (symbolAnalysisApplication *SymbolAnalysisApplication) GetAnalysisEvent(ctx context.Context, analysisEventID uint) (dto.AnalysisEventDto, error) {
	return symbolAnalysisApplication.symbolAnalysisService.GetAnalysisEvent(ctx, analysisEventID)
}

func (symbolAnalysisApplication *SymbolAnalysisApplication) FailInterruptedAnalysisEvents(ctx context.Context) error {
	return symbolAnalysisApplication.symbolAnalysisService.FailInterruptedAnalysisEvents(ctx)
}

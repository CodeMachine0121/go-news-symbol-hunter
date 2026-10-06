package application

import (
	"context"
	"log"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/service"
)

type SymbolAnalysisApplication struct {
	symbolAnalysisService *service.SymbolAnalysisService
	analysisSlots         chan struct{}
}

func NewSymbolAnalysisApplication(symbolAnalysisService *service.SymbolAnalysisService, maximumConcurrentAnalyses int) *SymbolAnalysisApplication {
	return &SymbolAnalysisApplication{symbolAnalysisService: symbolAnalysisService, analysisSlots: make(chan struct{}, maximumConcurrentAnalyses)}
}

func (symbolAnalysisApplication *SymbolAnalysisApplication) StartSymbolAnalysis(ctx context.Context, startSymbolAnalysisDto dto.StartSymbolAnalysisDto) (dto.StartedSymbolAnalysisDto, error) {
	hasAnalysisSlot := false
	select {
	case symbolAnalysisApplication.analysisSlots <- struct{}{}:
		hasAnalysisSlot = true
	default:
	}
	startSymbolAnalysisDto.AllowsNewAnalysis = hasAnalysisSlot
	startedSymbolAnalysis, err := symbolAnalysisApplication.symbolAnalysisService.StartSymbolAnalysis(ctx, startSymbolAnalysisDto)
	if err != nil || !startedSymbolAnalysis.IsNew {
		if hasAnalysisSlot {
			<-symbolAnalysisApplication.analysisSlots
		}
		return startedSymbolAnalysis, err
	}
	analyzeSymbolDto := dto.AnalyzeSymbolDto{AnalysisEventID: startedSymbolAnalysis.AnalysisEvent.AnalysisEventID, SearchKeyword: startedSymbolAnalysis.SearchKeyword}
	// the analysis outlives the HTTP request that started it and holds its slot until it finishes
	go func() {
		defer func() { <-symbolAnalysisApplication.analysisSlots }()
		symbolAnalysisApplication.AnalyzeSymbol(context.WithoutCancel(ctx), analyzeSymbolDto)
	}()
	return startedSymbolAnalysis, nil
}

func (symbolAnalysisApplication *SymbolAnalysisApplication) AnalyzeSymbol(ctx context.Context, analyzeSymbolDto dto.AnalyzeSymbolDto) {
	// a panicking analysis must not take down the server; its event is later failed as timed out
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("analysis event %d panicked: %v", analyzeSymbolDto.AnalysisEventID, recovered)
		}
	}()
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

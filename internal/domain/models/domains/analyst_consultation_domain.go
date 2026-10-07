package domains

import (
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

type AnalystConsultationDomain struct {
	conclusion     *AnalysisConclusionDomain
	failureReason  string
	usage          vo.AnalystUsageVo
	answeringModel string
}

func NewConcludedAnalystConsultationDomain(conclusion AnalysisConclusionDomain, usage vo.AnalystUsageVo, answeringModel string) AnalystConsultationDomain {
	return AnalystConsultationDomain{conclusion: &conclusion, usage: usage, answeringModel: answeringModel}
}

func NewFailedAnalystConsultationDomain(failureReason string, usage vo.AnalystUsageVo, answeringModel string) AnalystConsultationDomain {
	return AnalystConsultationDomain{failureReason: failureReason, usage: usage, answeringModel: answeringModel}
}

func (analystConsultationDomain AnalystConsultationDomain) Conclusion() (AnalysisConclusionDomain, bool) {
	if analystConsultationDomain.conclusion == nil {
		return AnalysisConclusionDomain{}, false
	}
	return *analystConsultationDomain.conclusion, true
}

func (analystConsultationDomain AnalystConsultationDomain) FailureReason() string {
	return analystConsultationDomain.failureReason
}

func (analystConsultationDomain AnalystConsultationDomain) Usage() vo.AnalystUsageVo {
	return analystConsultationDomain.usage
}

// empty when the analyst never answered
func (analystConsultationDomain AnalystConsultationDomain) AnsweringModel() string {
	return analystConsultationDomain.answeringModel
}

func (analystConsultationDomain AnalystConsultationDomain) ToSessionGradeEntity(symbol vo.SymbolVo, tradingDay string, session string, createdAt time.Time) (entities.SessionGrade, bool) {
	if analystConsultationDomain.conclusion == nil {
		return entities.SessionGrade{}, false
	}
	conclusion := analystConsultationDomain.conclusion
	return entities.SessionGrade{
		Symbol:       symbol.Value,
		Category:     symbol.Category.Value,
		TradingDay:   tradingDay,
		Session:      session,
		Grade:        conclusion.grade,
		Confidence:   conclusion.confidence,
		Reason:       conclusion.reason,
		KeyEvents:    conclusion.keyEvents,
		RiskFactors:  conclusion.riskFactors,
		Evidence:     conclusion.evidence,
		Model:        analystConsultationDomain.answeringModel,
		InputTokens:  analystConsultationDomain.usage.InputTokens,
		OutputTokens: analystConsultationDomain.usage.OutputTokens,
		CreatedAt:    createdAt,
	}, true
}

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
	sessionGrade := analystConsultationDomain.conclusion.ToSessionGradeEntity(symbol, tradingDay, session, createdAt)
	sessionGrade.Model = analystConsultationDomain.answeringModel
	sessionGrade.InputTokens = analystConsultationDomain.usage.InputTokens
	sessionGrade.OutputTokens = analystConsultationDomain.usage.OutputTokens
	return sessionGrade, true
}

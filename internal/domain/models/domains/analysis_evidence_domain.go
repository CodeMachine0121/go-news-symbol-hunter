package domains

import (
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
)

type AnalysisEvidenceDomain struct {
	evidence      []entities.AnalysisEvidence
	indexesByLink map[string]int
}

func NewAnalysisEvidenceDomain() *AnalysisEvidenceDomain {
	return &AnalysisEvidenceDomain{evidence: []entities.AnalysisEvidence{}, indexesByLink: map[string]int{}}
}

func (analysisEvidenceDomain *AnalysisEvidenceDomain) Record(news []dto.NewsDto) {
	for _, newsDto := range news {
		if _, recorded := analysisEvidenceDomain.indexesByLink[newsDto.Link]; recorded {
			continue
		}
		analysisEvidenceDomain.indexesByLink[newsDto.Link] = len(analysisEvidenceDomain.evidence)
		analysisEvidenceDomain.evidence = append(analysisEvidenceDomain.evidence, entities.AnalysisEvidence{
			Title:        newsDto.Title,
			Link:         newsDto.Link,
			PublishedAt:  newsDto.PublishedAt,
			ProviderName: newsDto.ProviderName,
		})
	}
}

func (analysisEvidenceDomain *AnalysisEvidenceDomain) FindByLink(link string) (entities.AnalysisEvidence, bool) {
	index, recorded := analysisEvidenceDomain.indexesByLink[link]
	if !recorded {
		return entities.AnalysisEvidence{}, false
	}
	return analysisEvidenceDomain.evidence[index], true
}

func (analysisEvidenceDomain *AnalysisEvidenceDomain) IsEmpty() bool {
	return len(analysisEvidenceDomain.evidence) == 0
}

func (analysisEvidenceDomain *AnalysisEvidenceDomain) ToEntities() []entities.AnalysisEvidence {
	return analysisEvidenceDomain.evidence
}

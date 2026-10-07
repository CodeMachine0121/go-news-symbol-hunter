package vo

import "time"

type AnalystRequestVo struct {
	Symbol        string
	Category      string
	SearchKeyword string
	// zero means the analyst is not limited to recent news
	PublishedSince time.Time
}

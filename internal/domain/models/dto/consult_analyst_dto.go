package dto

import "time"

type ConsultAnalystDto struct {
	Symbol        string
	Category      string
	SearchKeyword string
	// zero means the analyst may read every news item the news search keeps
	PublishedSince time.Time
}

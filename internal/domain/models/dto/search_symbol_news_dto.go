package dto

import "time"

type SearchSymbolNewsDto struct {
	Symbol   string
	Category string
	// zero keeps every news item inside the news search window
	PublishedSince time.Time
}

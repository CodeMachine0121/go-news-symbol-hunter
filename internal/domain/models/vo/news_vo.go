package vo

import "time"

type NewsVo struct {
	Title        string
	Link         string
	PublishedAt  time.Time
	ProviderName string
	Summary      string
}

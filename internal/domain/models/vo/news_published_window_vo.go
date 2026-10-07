package vo

import "time"

// Since is inclusive and Before exclusive; a zero bound leaves that side open
type NewsPublishedWindowVo struct {
	Since  time.Time
	Before time.Time
}

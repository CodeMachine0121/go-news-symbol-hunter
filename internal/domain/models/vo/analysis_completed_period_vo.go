package vo

import (
	"errors"
	"time"
)

const completedDateLayout = "2006-01-02"

var (
	ErrAnalysisCompletedPeriodInvalid = errors.New("期間不正確")
	taipeiTimeZone                    = time.FixedZone("Asia/Taipei", 8*60*60)
)

// From is inclusive and Before exclusive; a zero bound leaves that side open
type AnalysisCompletedPeriodVo struct {
	From   time.Time
	Before time.Time
}

// both dates are Taipei calendar days and both are included
func NewAnalysisCompletedPeriodVo(rawCompletedFrom string, rawCompletedTo string) (AnalysisCompletedPeriodVo, error) {
	completedPeriod := AnalysisCompletedPeriodVo{}
	if rawCompletedFrom != "" {
		completedFrom, err := time.ParseInLocation(completedDateLayout, rawCompletedFrom, taipeiTimeZone)
		if err != nil {
			return AnalysisCompletedPeriodVo{}, ErrAnalysisCompletedPeriodInvalid
		}
		completedPeriod.From = completedFrom
	}
	if rawCompletedTo != "" {
		completedTo, err := time.ParseInLocation(completedDateLayout, rawCompletedTo, taipeiTimeZone)
		if err != nil {
			return AnalysisCompletedPeriodVo{}, ErrAnalysisCompletedPeriodInvalid
		}
		completedPeriod.Before = completedTo.AddDate(0, 0, 1)
	}
	if !completedPeriod.From.IsZero() && !completedPeriod.Before.IsZero() && !completedPeriod.From.Before(completedPeriod.Before) {
		return AnalysisCompletedPeriodVo{}, ErrAnalysisCompletedPeriodInvalid
	}
	return completedPeriod, nil
}

package vo_test

import (
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalysisCompletedPeriodVo_CoversWholeTaipeiDays(t *testing.T) {
	taipei := time.FixedZone("Asia/Taipei", 8*60*60)
	testCases := []struct {
		name           string
		completedFrom  string
		completedTo    string
		expectedFrom   time.Time
		expectedBefore time.Time
	}{
		{name: "both days included", completedFrom: "2026-10-01", completedTo: "2026-10-31", expectedFrom: time.Date(2026, 10, 1, 0, 0, 0, 0, taipei), expectedBefore: time.Date(2026, 11, 1, 0, 0, 0, 0, taipei)},
		{name: "a single day", completedFrom: "2026-10-09", completedTo: "2026-10-09", expectedFrom: time.Date(2026, 10, 9, 0, 0, 0, 0, taipei), expectedBefore: time.Date(2026, 10, 10, 0, 0, 0, 0, taipei)},
		{name: "only a start", completedFrom: "2026-10-01", expectedFrom: time.Date(2026, 10, 1, 0, 0, 0, 0, taipei)},
		{name: "only an end", completedTo: "2026-10-31", expectedBefore: time.Date(2026, 11, 1, 0, 0, 0, 0, taipei)},
		{name: "no period at all"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			completedPeriod, err := vo.NewAnalysisCompletedPeriodVo(testCase.completedFrom, testCase.completedTo)

			require.NoError(t, err)
			assert.True(t, testCase.expectedFrom.Equal(completedPeriod.From), "from %v", completedPeriod.From)
			assert.True(t, testCase.expectedBefore.Equal(completedPeriod.Before), "before %v", completedPeriod.Before)
		})
	}
}

func TestAnalysisCompletedPeriodVo_RejectsAnUnreadableOrBackwardsPeriod(t *testing.T) {
	testCases := []struct {
		name          string
		completedFrom string
		completedTo   string
	}{
		{name: "start after end", completedFrom: "2026-10-31", completedTo: "2026-10-01"},
		{name: "start one day after end", completedFrom: "2026-10-02", completedTo: "2026-10-01"},
		{name: "unreadable start", completedFrom: "2026/10/01"},
		{name: "unreadable end", completedTo: "yesterday"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := vo.NewAnalysisCompletedPeriodVo(testCase.completedFrom, testCase.completedTo)

			assert.ErrorIs(t, err, vo.ErrAnalysisCompletedPeriodInvalid)
		})
	}
}

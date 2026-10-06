package utilities_test

import (
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/utilities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepublicOfChinaDateParser_ParsesTaipeiMidnight(t *testing.T) {
	parsedDate, err := utilities.NewRepublicOfChinaDateParser().ParseTaipeiMidnight(" 1151006 ")

	require.NoError(t, err)
	assert.True(t, parsedDate.Equal(time.Date(2026, 10, 5, 16, 0, 0, 0, time.UTC)))
	_, offset := parsedDate.Zone()
	assert.Equal(t, 8*60*60, offset)
}

func TestRepublicOfChinaDateParser_RejectsMalformedDates(t *testing.T) {
	for _, rawDate := range []string{"", "115", "ABC1005", "1151399"} {
		_, err := utilities.NewRepublicOfChinaDateParser().ParseTaipeiMidnight(rawDate)

		assert.Error(t, err, rawDate)
	}
}

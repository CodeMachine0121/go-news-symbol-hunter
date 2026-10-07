package vo_test

import (
	"testing"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/stretchr/testify/assert"
)

func TestNewSessionWeightsVo(t *testing.T) {
	testCases := []struct {
		name                       string
		preMarket, intraday, after float64
		expected                   vo.SessionWeightsVo
	}{
		{name: "configured weights are kept", preMarket: 0.4, intraday: 0.1, after: 0.6, expected: vo.SessionWeightsVo{PreMarket: 0.4, Intraday: 0.1, AfterMarket: 0.6}},
		{name: "zero after-market weight falls back to 0.5", preMarket: 0.3, intraday: 0.2, after: 0, expected: vo.SessionWeightsVo{PreMarket: 0.3, Intraday: 0.2, AfterMarket: 0.5}},
		{name: "zero pre-market and intraday weights fall back to the defaults", preMarket: 0, intraday: 0, after: 0.5, expected: vo.SessionWeightsVo{PreMarket: 0.3, Intraday: 0.2, AfterMarket: 0.5}},
		{name: "negative weights fall back to the defaults", preMarket: -1, intraday: -0.2, after: -3, expected: vo.SessionWeightsVo{PreMarket: 0.3, Intraday: 0.2, AfterMarket: 0.5}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, vo.NewSessionWeightsVo(testCase.preMarket, testCase.intraday, testCase.after))
		})
	}
}

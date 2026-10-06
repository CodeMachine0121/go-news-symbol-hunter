package vo_test

import (
	"strings"
	"testing"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/stretchr/testify/assert"
)

func TestNewApiKeyNameVo(t *testing.T) {
	testCases := []struct {
		name          string
		rawName       string
		expectedValue string
		expectedError error
	}{
		{name: "keeps a valid name", rawName: "我的研究腳本", expectedValue: "我的研究腳本"},
		{name: "trims surrounding spaces", rawName: "  研究  ", expectedValue: "研究"},
		{name: "accepts exactly 100 characters", rawName: strings.Repeat("研", 100), expectedValue: strings.Repeat("研", 100)},
		{name: "rejects 101 characters", rawName: strings.Repeat("研", 101), expectedError: vo.ErrApiKeyNameTooLong},
		{name: "rejects an empty name", rawName: "", expectedError: vo.ErrApiKeyNameRequired},
		{name: "rejects a blank name", rawName: "   ", expectedError: vo.ErrApiKeyNameRequired},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			name, err := vo.NewApiKeyNameVo(testCase.rawName)

			assert.ErrorIs(t, err, testCase.expectedError)
			assert.Equal(t, testCase.expectedValue, name.Value)
		})
	}
}

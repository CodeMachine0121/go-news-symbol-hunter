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
		{name: "rejects a NUL character", rawName: "研\x00究", expectedError: vo.ErrApiKeyNameInvalidCharacters},
		{name: "rejects an inner tab", rawName: "研\t究", expectedError: vo.ErrApiKeyNameInvalidCharacters},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			name, err := vo.NewApiKeyNameVo(testCase.rawName)

			assert.ErrorIs(t, err, testCase.expectedError)
			assert.Equal(t, testCase.expectedValue, name.Value)
		})
	}
}

func TestApiKeyNameErrors_StateTheBusinessMessages(t *testing.T) {
	assert.EqualError(t, vo.ErrApiKeyNameRequired, "API key 名稱為必填")
	assert.EqualError(t, vo.ErrApiKeyNameTooLong, "API key 名稱不可超過 100 個字")
	assert.EqualError(t, vo.ErrApiKeyNameInvalidCharacters, "API key 名稱不可包含控制字元")
}

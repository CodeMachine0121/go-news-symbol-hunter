package vo

import (
	"errors"
	"strings"
	"unicode/utf8"
)

const apiKeyNameMaximumLength = 100

var (
	ErrApiKeyNameRequired = errors.New("API key 名稱為必填")
	ErrApiKeyNameTooLong  = errors.New("API key 名稱不可超過 100 個字")
)

type ApiKeyNameVo struct {
	Value string
}

func NewApiKeyNameVo(rawName string) (ApiKeyNameVo, error) {
	trimmedName := strings.TrimSpace(rawName)
	if trimmedName == "" {
		return ApiKeyNameVo{}, ErrApiKeyNameRequired
	}
	if utf8.RuneCountInString(trimmedName) > apiKeyNameMaximumLength {
		return ApiKeyNameVo{}, ErrApiKeyNameTooLong
	}
	return ApiKeyNameVo{Value: trimmedName}, nil
}

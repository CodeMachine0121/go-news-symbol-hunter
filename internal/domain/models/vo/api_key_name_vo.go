package vo

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const ApiKeyNameMaximumLength = 100

var (
	ErrApiKeyNameRequired          = errors.New("API key 名稱為必填")
	ErrApiKeyNameTooLong           = fmt.Errorf("API key 名稱不可超過 %d 個字", ApiKeyNameMaximumLength)
	ErrApiKeyNameInvalidCharacters = errors.New("API key 名稱不可包含控制字元")
)

type ApiKeyNameVo struct {
	Value string
}

func NewApiKeyNameVo(rawName string) (ApiKeyNameVo, error) {
	trimmedName := strings.TrimSpace(rawName)
	if trimmedName == "" {
		return ApiKeyNameVo{}, ErrApiKeyNameRequired
	}
	if utf8.RuneCountInString(trimmedName) > ApiKeyNameMaximumLength {
		return ApiKeyNameVo{}, ErrApiKeyNameTooLong
	}
	if strings.ContainsFunc(trimmedName, unicode.IsControl) {
		return ApiKeyNameVo{}, ErrApiKeyNameInvalidCharacters
	}
	return ApiKeyNameVo{Value: trimmedName}, nil
}

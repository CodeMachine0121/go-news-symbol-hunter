package service

import (
	"errors"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

var (
	ErrApiKeyNameRequired          = vo.ErrApiKeyNameRequired
	ErrApiKeyNameTooLong           = vo.ErrApiKeyNameTooLong
	ErrApiKeyNameInvalidCharacters = vo.ErrApiKeyNameInvalidCharacters
	ErrApiKeyMissing               = vo.ErrApiKeyMissing
	ErrApiKeyInvalid               = domains.ErrApiKeyInvalid
	ErrApiKeyInactive              = domains.ErrApiKeyInactive
	ErrApiKeyStorageUnavailable    = errors.New("API key storage unavailable")
)

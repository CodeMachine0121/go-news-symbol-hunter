package vo_test

import (
	"testing"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateApiKeySecretVo_ProducesDistinctPrefixedSecrets(t *testing.T) {
	firstSecret := vo.GenerateApiKeySecretVo()
	secondSecret := vo.GenerateApiKeySecretVo()

	assert.Regexp(t, `^snh_[A-Za-z0-9_-]{43}$`, firstSecret.Plaintext)
	assert.NotEqual(t, firstSecret.Plaintext, secondSecret.Plaintext)
	presentedSecret, err := vo.NewApiKeySecretVo(firstSecret.Plaintext)
	require.NoError(t, err)
	assert.Equal(t, presentedSecret.Hash, firstSecret.Hash)
}

func TestNewApiKeySecretVo(t *testing.T) {
	testCases := []struct {
		name              string
		presentedSecret   string
		expectedPlaintext string
		expectedHash      string
		expectedError     error
	}{
		{name: "hashes the presented secret with SHA-256", presentedSecret: "abc", expectedPlaintext: "abc", expectedHash: "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"},
		{name: "rejects a missing secret", presentedSecret: "", expectedError: vo.ErrApiKeyMissing},
		{name: "rejects a blank secret", presentedSecret: "  ", expectedError: vo.ErrApiKeyMissing},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			secret, err := vo.NewApiKeySecretVo(testCase.presentedSecret)

			assert.ErrorIs(t, err, testCase.expectedError)
			assert.Equal(t, testCase.expectedPlaintext, secret.Plaintext)
			assert.Equal(t, testCase.expectedHash, secret.Hash)
		})
	}
}

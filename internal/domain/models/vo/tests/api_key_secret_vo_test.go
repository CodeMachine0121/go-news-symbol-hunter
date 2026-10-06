package vo_test

import (
	"testing"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/stretchr/testify/assert"
)

func TestNewIssuedApiKeySecretVo_PrefixesTheEncodedRandomBytes(t *testing.T) {
	secret := vo.NewIssuedApiKeySecretVo(make([]byte, vo.ApiKeySecretRandomByteLength))

	assert.Equal(t, "snh_AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", secret.Plaintext)
	assert.Equal(t, 32, vo.ApiKeySecretRandomByteLength)
}

func TestNewIssuedApiKeySecretVo_HashesLikeAPresentedSecret(t *testing.T) {
	issuedSecret := vo.NewIssuedApiKeySecretVo([]byte{1, 2, 3})

	presentedSecret, err := vo.NewApiKeySecretVo(issuedSecret.Plaintext)

	assert.NoError(t, err)
	assert.Equal(t, presentedSecret.Hash, issuedSecret.Hash)
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
		{name: "trims the presented secret", presentedSecret: " abc ", expectedPlaintext: "abc", expectedHash: "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"},
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

package system_test

import (
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/system"
	"github.com/stretchr/testify/assert"
)

func TestSystemClockProxy_ReturnsTheCurrentTime(t *testing.T) {
	before := time.Now()

	now := system.NewSystemClockProxy().Now()

	assert.False(t, now.Before(before))
	assert.False(t, now.After(time.Now()))
}

func TestCryptoRandomProxy_GeneratesDistinctBytesOfTheRequestedLength(t *testing.T) {
	cryptoRandomProxy := system.NewCryptoRandomProxy()

	firstBytes := cryptoRandomProxy.GenerateBytes(32)
	secondBytes := cryptoRandomProxy.GenerateBytes(32)

	assert.Len(t, firstBytes, 32)
	assert.NotEqual(t, firstBytes, secondBytes)
}

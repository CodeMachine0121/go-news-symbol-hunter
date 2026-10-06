package utilities_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/utilities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHttpBodyReader_ReturnsTheBodyOfASuccessfulResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.NotEmpty(t, request.Header.Get("User-Agent"))
		_, _ = writer.Write([]byte("feed"))
	}))
	defer server.Close()

	body, err := utilities.NewHttpBodyReader(server.Client()).Read(server.URL)

	require.NoError(t, err)
	assert.Equal(t, "feed", string(body))
}

func TestHttpBodyReader_RejectsNonSuccessStatuses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	_, err := utilities.NewHttpBodyReader(server.Client()).Read(server.URL)

	assert.ErrorContains(t, err, "unexpected status 429")
}

func TestHttpBodyReader_ReportsUnreachableServers(t *testing.T) {
	_, err := utilities.NewHttpBodyReader(http.DefaultClient).Read("http://127.0.0.1:1")

	assert.Error(t, err)
}

func TestHttpBodyReader_RejectsInvalidUrls(t *testing.T) {
	_, err := utilities.NewHttpBodyReader(http.DefaultClient).Read("http://bad host")

	assert.Error(t, err)
}

func TestHttpBodyReader_FailsWhenTheSourceExceedsTheClientTimeout(t *testing.T) {
	releaseServer := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		<-releaseServer
	}))
	defer server.Close()
	defer close(releaseServer)
	timeoutClient := server.Client()
	timeoutClient.Timeout = 50 * time.Millisecond

	_, err := utilities.NewHttpBodyReader(timeoutClient).Read(server.URL)

	assert.ErrorContains(t, err, "Timeout")
}

package httpfetch_test

import (
	"context"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/httpfetch"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHttpBodyReader_ReturnsTheBodyOfASuccessfulResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.NotEmpty(t, request.Header.Get("User-Agent"))
		_, _ = writer.Write([]byte("feed"))
	}))
	defer server.Close()

	body, err := httpfetch.NewHttpBodyReader(server.Client()).Read(context.Background(), server.URL)

	require.NoError(t, err)
	assert.Equal(t, "feed", string(body))
}

func TestHttpBodyReader_RejectsNonSuccessStatuses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	_, err := httpfetch.NewHttpBodyReader(server.Client()).Read(context.Background(), server.URL)

	assert.ErrorContains(t, err, "unexpected status 429")
}

func TestHttpBodyReader_ReportsUnreachableServers(t *testing.T) {
	_, err := httpfetch.NewHttpBodyReader(http.DefaultClient).Read(context.Background(), "http://127.0.0.1:1")

	assert.Error(t, err)
}

func TestHttpBodyReader_RejectsInvalidUrls(t *testing.T) {
	_, err := httpfetch.NewHttpBodyReader(http.DefaultClient).Read(context.Background(), "http://bad host")

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

	_, err := httpfetch.NewHttpBodyReader(timeoutClient).Read(context.Background(), server.URL)

	assert.ErrorContains(t, err, "Timeout")
}

func TestHttpBodyReader_RejectsOversizedBodies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(strings.Repeat("x", 10<<20+1)))
	}))
	defer server.Close()

	_, err := httpfetch.NewHttpBodyReader(server.Client()).Read(context.Background(), server.URL)

	assert.ErrorContains(t, err, "exceeds")
}

func TestHttpBodyReader_AcceptsABodyAtTheLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(strings.Repeat("x", 10<<20)))
	}))
	defer server.Close()

	body, err := httpfetch.NewHttpBodyReader(server.Client()).Read(context.Background(), server.URL)

	require.NoError(t, err)
	assert.Len(t, body, 10<<20)
}

func TestHttpBodyReader_StopsWhenTheCallerCancels(t *testing.T) {
	releaseServer := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		<-releaseServer
	}))
	defer server.Close()
	defer close(releaseServer)
	cancelledContext, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := httpfetch.NewHttpBodyReader(server.Client()).Read(cancelledContext, server.URL)

	assert.ErrorIs(t, err, context.Canceled)
}

func TestHttpBodyReader_ReportsABodyCutOffMidway(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Length", "100")
		_, _ = writer.Write([]byte("partial"))
		connection, _, err := writer.(http.Hijacker).Hijack()
		require.NoError(t, err)
		_ = connection.Close()
	}))
	defer server.Close()

	_, err := httpfetch.NewHttpBodyReader(server.Client()).Read(context.Background(), server.URL)

	assert.Error(t, err)
}

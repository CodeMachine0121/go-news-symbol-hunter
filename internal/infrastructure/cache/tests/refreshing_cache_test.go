package cache_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface/mocks"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var cachedAt = time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

func TestRefreshingCache_DownloadsAgainOnlyAfterItExpires(t *testing.T) {
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(cachedAt).Once()
	clockProxy.EXPECT().Now().Return(cachedAt.Add(time.Hour - time.Second)).Once()
	clockProxy.EXPECT().Now().Return(cachedAt.Add(time.Hour)).Once()
	downloadCount := 0
	refreshingCache := cache.NewRefreshingCache(clockProxy, time.Hour, func(context.Context) (int, error) {
		downloadCount++
		return downloadCount, nil
	})

	firstValue, _ := refreshingCache.Get(context.Background())
	cachedValue, _ := refreshingCache.Get(context.Background())
	refreshedValue, err := refreshingCache.Get(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []int{1, 1, 2}, []int{firstValue, cachedValue, refreshedValue})
}

func TestRefreshingCache_DoesNotCacheAFailedDownload(t *testing.T) {
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(cachedAt)
	downloadCount := 0
	refreshingCache := cache.NewRefreshingCache(clockProxy, time.Hour, func(context.Context) (string, error) {
		downloadCount++
		if downloadCount == 1 {
			return "", errors.New("source down")
		}
		return "value", nil
	})

	_, firstError := refreshingCache.Get(context.Background())
	value, secondError := refreshingCache.Get(context.Background())

	assert.Error(t, firstError)
	require.NoError(t, secondError)
	assert.Equal(t, "value", value)
}

func TestRefreshingCache_SharesOneDownloadAcrossConcurrentCallers(t *testing.T) {
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(cachedAt)
	releaseDownload := make(chan struct{})
	downloadCount := &atomic.Int32{}
	refreshingCache := cache.NewRefreshingCache(clockProxy, time.Hour, func(context.Context) (string, error) {
		downloadCount.Add(1)
		<-releaseDownload
		return "value", nil
	})
	values := make([]string, 5)
	var waitGroup sync.WaitGroup
	for index := range values {
		waitGroup.Go(func() { values[index], _ = refreshingCache.Get(context.Background()) })
	}
	time.Sleep(50 * time.Millisecond)

	close(releaseDownload)
	waitGroup.Wait()

	assert.Equal(t, int32(1), downloadCount.Load())
	assert.Equal(t, []string{"value", "value", "value", "value", "value"}, values)
}

func TestRefreshingCache_KeepsServingWhileARefreshIsInFlight(t *testing.T) {
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(cachedAt).Once()
	clockProxy.EXPECT().Now().Return(cachedAt.Add(2 * time.Hour)).Once()
	clockProxy.EXPECT().Now().Return(cachedAt.Add(time.Minute))
	releaseRefresh := make(chan struct{})
	downloadCount := &atomic.Int32{}
	refreshingCache := cache.NewRefreshingCache(clockProxy, time.Hour, func(context.Context) (int32, error) {
		if downloadCount.Add(1) == 2 {
			<-releaseRefresh
		}
		return downloadCount.Load(), nil
	})
	_, _ = refreshingCache.Get(context.Background())
	refreshDone := make(chan struct{})
	go func() {
		_, _ = refreshingCache.Get(context.Background())
		close(refreshDone)
	}()
	time.Sleep(50 * time.Millisecond)

	value, err := refreshingCache.Get(context.Background())

	close(releaseRefresh)
	<-refreshDone
	require.NoError(t, err)
	assert.Equal(t, int32(1), value)
}

func TestRefreshingCache_DownloadSurvivesTheCallerCancelling(t *testing.T) {
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(cachedAt)
	refreshingCache := cache.NewRefreshingCache(clockProxy, time.Hour, func(ctx context.Context) (error, error) {
		return ctx.Err(), nil
	})
	cancelledContext, cancel := context.WithCancel(context.Background())
	cancel()

	downloadContextError, err := refreshingCache.Get(cancelledContext)

	require.NoError(t, err)
	assert.NoError(t, downloadContextError)
}

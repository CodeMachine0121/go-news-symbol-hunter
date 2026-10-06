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
	clockProxy.EXPECT().Now().Return(cachedAt).Times(2)
	clockProxy.EXPECT().Now().Return(cachedAt.Add(time.Hour - time.Second)).Once()
	clockProxy.EXPECT().Now().Return(cachedAt.Add(time.Hour))
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
	clockProxy.EXPECT().Now().Return(cachedAt).Times(2)
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
	releaseDownload := make(chan struct{})
	downloaded := make(chan struct{})
	downloadCount := &atomic.Int32{}
	refreshingCache := cache.NewRefreshingCache(clockProxy, time.Hour, func(ctx context.Context) (error, error) {
		downloadCount.Add(1)
		<-releaseDownload
		defer close(downloaded)
		return ctx.Err(), nil
	})
	cancelledContext, cancel := context.WithCancel(context.Background())
	cancel()

	_, cancelledError := refreshingCache.Get(cancelledContext)
	close(releaseDownload)
	<-downloaded
	time.Sleep(10 * time.Millisecond)
	downloadContextError, err := refreshingCache.Get(context.Background())

	assert.ErrorIs(t, cancelledError, context.Canceled)
	require.NoError(t, err)
	assert.NoError(t, downloadContextError)
	assert.Equal(t, int32(1), downloadCount.Load())
}

func TestRefreshingCache_KeepsTheLastGoodValueWhenARefreshFails(t *testing.T) {
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(cachedAt).Times(2)
	clockProxy.EXPECT().Now().Return(cachedAt.Add(2 * time.Hour))
	downloadCount := 0
	refreshingCache := cache.NewRefreshingCache(clockProxy, time.Hour, func(context.Context) (string, error) {
		downloadCount++
		if downloadCount > 1 {
			return "", errors.New("source down")
		}
		return "yesterday", nil
	})
	_, _ = refreshingCache.Get(context.Background())

	value, err := refreshingCache.Get(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "yesterday", value)
	assert.Equal(t, 2, downloadCount)
}

func TestRefreshingCache_StopsWaitingWhenTheCallerGivesUp(t *testing.T) {
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(cachedAt).Maybe()
	releaseDownload := make(chan struct{})
	defer close(releaseDownload)
	refreshingCache := cache.NewRefreshingCache(clockProxy, time.Hour, func(context.Context) (string, error) {
		<-releaseDownload
		return "late", nil
	})
	shortContext, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	startedAt := time.Now()
	_, err := refreshingCache.Get(shortContext)

	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(startedAt), time.Second)
}

func TestRefreshingCache_ServesTheLastGoodValueWhenTheCallerGivesUp(t *testing.T) {
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(cachedAt).Times(2)
	clockProxy.EXPECT().Now().Return(cachedAt.Add(2 * time.Hour))
	releaseRefresh := make(chan struct{})
	defer close(releaseRefresh)
	downloadCount := 0
	refreshingCache := cache.NewRefreshingCache(clockProxy, time.Hour, func(context.Context) (string, error) {
		downloadCount++
		if downloadCount > 1 {
			<-releaseRefresh
		}
		return "yesterday", nil
	})
	_, _ = refreshingCache.Get(context.Background())
	shortContext, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	value, err := refreshingCache.Get(shortContext)

	require.NoError(t, err)
	assert.Equal(t, "yesterday", value)
}

func TestRefreshingCache_StartsFreshnessWhenTheDownloadFinishes(t *testing.T) {
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(cachedAt).Once()
	clockProxy.EXPECT().Now().Return(cachedAt.Add(30 * time.Minute)).Once()
	clockProxy.EXPECT().Now().Return(cachedAt.Add(time.Hour + 29*time.Minute)).Once()
	downloadCount := 0
	refreshingCache := cache.NewRefreshingCache(clockProxy, time.Hour, func(context.Context) (int, error) {
		downloadCount++
		return downloadCount, nil
	})

	_, _ = refreshingCache.Get(context.Background())
	value, _ := refreshingCache.Get(context.Background())

	assert.Equal(t, 1, value)
	assert.Equal(t, 1, downloadCount)
}

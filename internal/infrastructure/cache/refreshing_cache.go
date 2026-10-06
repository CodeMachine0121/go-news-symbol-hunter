package cache

import (
	"context"
	"sync"
	"time"

	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"golang.org/x/sync/singleflight"
)

type RefreshingCache[T any] struct {
	clockProxy   interfaces.IClockProxy
	freshFor     time.Duration
	download     func(ctx context.Context) (T, error)
	refreshGroup singleflight.Group
	cacheMutex   sync.RWMutex
	cachedValue  T
	hasValue     bool
	expiresAt    time.Time
}

func NewRefreshingCache[T any](clockProxy interfaces.IClockProxy, freshFor time.Duration, download func(ctx context.Context) (T, error)) *RefreshingCache[T] {
	return &RefreshingCache[T]{clockProxy: clockProxy, freshFor: freshFor, download: download}
}

// a failed refresh keeps serving the last good value; with none yet, the error is returned and nothing is cached
func (refreshingCache *RefreshingCache[T]) Get(ctx context.Context) (T, error) {
	cachedValue, hasValue, isFresh := refreshingCache.snapshot(refreshingCache.clockProxy.Now())
	if isFresh {
		return cachedValue, nil
	}
	// one download serves every concurrent caller and outlives any single caller's cancellation
	refreshResult := refreshingCache.refreshGroup.DoChan("refresh", func() (any, error) {
		downloadedValue, err := refreshingCache.download(context.WithoutCancel(ctx))
		if err != nil {
			return nil, err
		}
		refreshingCache.store(downloadedValue, refreshingCache.clockProxy.Now())
		return downloadedValue, nil
	})
	select {
	case result := <-refreshResult:
		if result.Err != nil && hasValue {
			return cachedValue, nil
		}
		if result.Err != nil {
			return cachedValue, result.Err
		}
		// comma-ok because singleflight boxes the value in an interface, which is nil when T is a nil interface
		refreshedValue, _ := result.Val.(T)
		return refreshedValue, nil
	case <-ctx.Done():
		if hasValue {
			return cachedValue, nil
		}
		return cachedValue, ctx.Err()
	}
}

// scopes the read lock so it is released before any network call
func (refreshingCache *RefreshingCache[T]) snapshot(now time.Time) (T, bool, bool) {
	refreshingCache.cacheMutex.RLock()
	defer refreshingCache.cacheMutex.RUnlock()
	return refreshingCache.cachedValue, refreshingCache.hasValue, refreshingCache.hasValue && now.Before(refreshingCache.expiresAt)
}

// scopes the write lock to the assignment
func (refreshingCache *RefreshingCache[T]) store(value T, downloadedAt time.Time) {
	refreshingCache.cacheMutex.Lock()
	defer refreshingCache.cacheMutex.Unlock()
	refreshingCache.cachedValue = value
	refreshingCache.hasValue = true
	refreshingCache.expiresAt = downloadedAt.Add(refreshingCache.freshFor)
}

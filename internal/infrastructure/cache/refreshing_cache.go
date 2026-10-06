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

// a failed download is not cached, so the next caller retries
func (refreshingCache *RefreshingCache[T]) Get(ctx context.Context) (T, error) {
	now := refreshingCache.clockProxy.Now()
	if cachedValue, isFresh := refreshingCache.freshValue(now); isFresh {
		return cachedValue, nil
	}
	// one download serves every concurrent caller; it outlives a single caller's cancellation
	downloadedValue, err, _ := refreshingCache.refreshGroup.Do("refresh", func() (any, error) {
		return refreshingCache.download(context.WithoutCancel(ctx))
	})
	if err != nil {
		var noValue T
		return noValue, err
	}
	// comma-ok because singleflight boxes the value in an interface, which is nil when T is a nil interface
	value, _ := downloadedValue.(T)
	refreshingCache.store(value, now)
	return value, nil
}

// scopes the read lock so it is released before any network call
func (refreshingCache *RefreshingCache[T]) freshValue(now time.Time) (T, bool) {
	refreshingCache.cacheMutex.RLock()
	defer refreshingCache.cacheMutex.RUnlock()
	return refreshingCache.cachedValue, refreshingCache.hasValue && now.Before(refreshingCache.expiresAt)
}

// scopes the write lock to the assignment
func (refreshingCache *RefreshingCache[T]) store(value T, now time.Time) {
	refreshingCache.cacheMutex.Lock()
	defer refreshingCache.cacheMutex.Unlock()
	refreshingCache.cachedValue = value
	refreshingCache.hasValue = true
	refreshingCache.expiresAt = now.Add(refreshingCache.freshFor)
}

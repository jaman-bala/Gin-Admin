package redis

// Note: This is an in-memory cache implementation, not a real Redis client.
// It's named "redis" for interface compatibility but uses map storage.
// TODO: Replace with actual Redis client for production.

import (
	"context"
	"fmt"
	"gin_auth_service/config"
	"gin_auth_service/internal/domain/token"
	"sync"
	"time"
)

type cacheItem struct {
	value      string
	expiration time.Time
}

type redisCache struct {
	store map[string]cacheItem
	mutex sync.RWMutex
}

// NewCache creates a new instance of Redis-like cache.
func NewCache(cfg *config.Config) (token.Repository, error) {
	if cfg.Redis.Host == "" {
		return nil, fmt.Errorf("redis host is required")
	}

	cache := &redisCache{
		store: make(map[string]cacheItem),
	}

	go cache.cleanupExpired()

	return cache, nil
}

func (r *redisCache) Set(_ context.Context, key string, value interface{}, expiration time.Duration) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.store[key] = cacheItem{
		value:      fmt.Sprintf("%v", value),
		expiration: time.Now().Add(expiration),
	}
	return nil
}

func (r *redisCache) Get(_ context.Context, key string) (string, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	item, exists := r.store[key]
	if !exists {
		return "", fmt.Errorf("key not found")
	}

	if time.Now().After(item.expiration) {
		delete(r.store, key)
		return "", fmt.Errorf("key expired")
	}

	return item.value, nil
}

func (r *redisCache) Delete(_ context.Context, key string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.store, key)
	return nil
}

func (r *redisCache) Exists(_ context.Context, key string) (bool, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	item, exists := r.store[key]
	if !exists {
		return false, nil
	}

	if time.Now().After(item.expiration) {
		delete(r.store, key)
		return false, nil
	}

	return true, nil
}

func (r *redisCache) cleanupExpired() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		r.mutex.Lock()
		now := time.Now()
		for key, item := range r.store {
			if now.After(item.expiration) {
				delete(r.store, key)
			}
		}
		r.mutex.Unlock()
	}
}

package redis

import (
	"context"
	"fmt"
	"time"

	"gin_auth_service/config"
	"gin_auth_service/internal/domain/token"

	goredis "github.com/redis/go-redis/v9"
)

// Cache wraps a Redis client. It implements token.Repository (blacklist)
// and middleware.RateLimiter (login rate limiting) via the same connection.
type Cache struct {
	client *goredis.Client
}

// incrementScript atomically increments a counter and sets its TTL on first creation.
// Using a Lua script guarantees both operations execute as a single Redis transaction,
// eliminating the race between INCR and EXPIRE that exists when done as two commands.
var incrementScript = goredis.NewScript(`
local n = redis.call('INCR', KEYS[1])
if n == 1 then
    redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return n
`)

// releaseScript gives one slot back (DECR) but only while the counter exists,
// and deletes it at zero. A bare DECR on a key that already expired would
// create it as -1 with no TTL: that counter would then live forever and
// silently grant extra attempts.
var releaseScript = goredis.NewScript(`
if redis.call('EXISTS', KEYS[1]) == 0 then
    return 0
end
local n = redis.call('DECR', KEYS[1])
if n <= 0 then
    redis.call('DEL', KEYS[1])
    return 0
end
return n
`)

// NewCache connects to Redis and verifies connectivity with a Ping.
func NewCache(cfg *config.Config) (*Cache, error) {
	if cfg.Redis.Host == "" {
		return nil, fmt.Errorf("redis host is required")
	}

	client := goredis.NewClient(&goredis.Options{
		Addr:     cfg.Redis.Host + ":" + cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &Cache{client: client}, nil
}

// --- token.Repository ---

func (c *Cache) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	return c.client.Set(ctx, key, value, expiration).Err()
}

func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err == goredis.Nil {
		return "", fmt.Errorf("key not found")
	}
	return val, err
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, key).Result()
	return n > 0, err
}

// --- middleware.RateLimiter ---

// Increment atomically increments the counter for key and sets window TTL on first call.
// Returns the counter value after increment.
func (c *Cache) Increment(ctx context.Context, key string, window time.Duration) (int64, error) {
	return incrementScript.Run(ctx, c.client, []string{key}, window.Milliseconds()).Int64()
}

// Release gives back one slot taken by Increment. Used when an attempt
// succeeded: only that attempt's own slot is returned, so failed attempts keep
// counting and one success can never wipe an attacker's failure history.
func (c *Cache) Release(ctx context.Context, key string) error {
	return releaseScript.Run(ctx, c.client, []string{key}).Err()
}

// Ping checks Redis connectivity.
func (c *Cache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Verify *Cache satisfies token.Repository at compile time.
var _ token.Repository = (*Cache)(nil)

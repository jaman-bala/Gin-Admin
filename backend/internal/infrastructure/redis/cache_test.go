package redis

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"gin_auth_service/config"
)

// Integration tests against a real Redis. Run with e.g.:
//
//	REDIS_TEST_ADDR=localhost:6379 go test ./internal/infrastructure/redis/...
//
// They use unique keys and clean up after themselves.
func newTestCache(t *testing.T) *Cache {
	t.Helper()
	addr := os.Getenv("REDIS_TEST_ADDR")
	if addr == "" {
		t.Skip("REDIS_TEST_ADDR not set; skipping Redis integration test")
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("REDIS_TEST_ADDR %q: %v", addr, err)
	}
	c, err := NewCache(&config.Config{Redis: config.RedisConfig{Host: host, Port: port}})
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}
	return c
}

func testKey(t *testing.T, c *Cache) string {
	t.Helper()
	key := "test:ratelimit:" + t.Name() + ":" + time.Now().Format("150405.000000000")
	t.Cleanup(func() { _ = c.Delete(context.Background(), key) })
	return key
}

func ttl(t *testing.T, c *Cache, key string) time.Duration {
	t.Helper()
	d, err := c.client.PTTL(context.Background(), key).Result()
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestIncrement_SetsTTLOnFirstCallOnly(t *testing.T) {
	c := newTestCache(t)
	ctx, key := context.Background(), testKey(t, c)

	if n, err := c.Increment(ctx, key, time.Minute); err != nil || n != 1 {
		t.Fatalf("first Increment = %d, %v; want 1", n, err)
	}
	first := ttl(t, c, key)
	if first <= 0 || first > time.Minute {
		t.Fatalf("ttl after first Increment = %v, want within (0, 1m]", first)
	}
	time.Sleep(20 * time.Millisecond)
	if n, _ := c.Increment(ctx, key, time.Hour); n != 2 {
		t.Fatalf("second Increment = %d, want 2", n)
	}
	if second := ttl(t, c, key); second > first {
		t.Errorf("ttl grew from %v to %v: later attempts must not extend the window", first, second)
	}
}

// Release returns exactly one slot: the other attempts keep counting.
func TestRelease_ReturnsOnlyItsOwnSlot(t *testing.T) {
	c := newTestCache(t)
	ctx, key := context.Background(), testKey(t, c)

	for i := 0; i < 5; i++ {
		if _, err := c.Increment(ctx, key, time.Minute); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.Release(ctx, key); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if n, _ := c.Increment(ctx, key, time.Minute); n != 5 {
		t.Fatalf("after 5 takes and 1 release, next Increment = %d, want 5", n)
	}
	if d := ttl(t, c, key); d <= 0 {
		t.Errorf("ttl = %v: Release must not strip the window", d)
	}
}

func TestRelease_DropsTheKeyAtZero(t *testing.T) {
	c := newTestCache(t)
	ctx, key := context.Background(), testKey(t, c)

	if _, err := c.Increment(ctx, key, time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := c.Release(ctx, key); err != nil {
		t.Fatal(err)
	}
	if ok, err := c.Exists(ctx, key); err != nil || ok {
		t.Fatalf("Exists = %v, %v; want the key gone", ok, err)
	}
}

// A bare DECR on a missing key would create it as -1 with NO expiry: the
// counter would live forever and quietly grant extra attempts.
func TestRelease_NeverCreatesOrGoesNegativeOnAMissingKey(t *testing.T) {
	c := newTestCache(t)
	ctx, key := context.Background(), testKey(t, c)

	if err := c.Release(ctx, key); err != nil {
		t.Fatalf("Release on a missing key: %v", err)
	}
	if ok, _ := c.Exists(ctx, key); ok {
		t.Fatal("Release created a key that did not exist")
	}
	if n, _ := c.Increment(ctx, key, time.Minute); n != 1 {
		t.Fatalf("Increment after a stray Release = %d, want 1", n)
	}
}

func TestRelease_AfterWindowExpiredDoesNotLeaveAPermanentCounter(t *testing.T) {
	c := newTestCache(t)
	ctx, key := context.Background(), testKey(t, c)

	if _, err := c.Increment(ctx, key, 50*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(120 * time.Millisecond) // window over, key expired
	if err := c.Release(ctx, key); err != nil {
		t.Fatal(err)
	}
	if n, _ := c.Increment(ctx, key, time.Minute); n != 1 {
		t.Fatalf("Increment = %d, want 1 (a fresh window)", n)
	}
	if d := ttl(t, c, key); d <= 0 {
		t.Fatalf("ttl = %v: the counter has no expiry", d)
	}
}

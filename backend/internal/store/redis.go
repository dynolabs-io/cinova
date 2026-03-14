package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// searchCacheTTL is the duration for which NL search results are cached.
	searchCacheTTL = time.Hour

	// rateLimitWindow is the sliding window for rate limiting.
	rateLimitWindow = time.Minute

	// rateLimitMax is the maximum requests per session per window.
	rateLimitMax = 30
)

// RedisStore wraps a Redis/Valkey client and provides caching and rate-limiting.
type RedisStore struct {
	client *redis.Client
}

// RedisClient is an alias for RedisStore for ergonomic use in other packages.
type RedisClient = RedisStore

// NewRedisStore connects to Redis/Valkey using the given URL.
func NewRedisStore(ctx context.Context, redisURL string) (*RedisStore, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("redis parse url: %w", err)
	}
	opts.PoolSize = 10
	opts.MinIdleConns = 2
	opts.ConnMaxLifetime = 30 * time.Minute

	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &RedisStore{client: client}, nil
}

// Close releases the Redis connection pool.
func (r *RedisStore) Close() error {
	return r.client.Close()
}

// Ping checks Redis connectivity.
func (r *RedisStore) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// ---- Search result cache ----

// CacheSearchResults stores marshalled search results under a key derived from
// the query and country. TTL is searchCacheTTL (1 hour).
func (r *RedisStore) CacheSearchResults(ctx context.Context, query, country string, results interface{}) error {
	key := searchCacheKey(query, country)
	data, err := json.Marshal(results)
	if err != nil {
		return fmt.Errorf("marshal search results: %w", err)
	}
	return r.client.Set(ctx, key, data, searchCacheTTL).Err()
}

// GetCachedSearchResults retrieves previously cached search results. Returns
// (nil, nil) on cache miss.
func (r *RedisStore) GetCachedSearchResults(ctx context.Context, query, country string) ([]byte, error) {
	key := searchCacheKey(query, country)
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get cached search: %w", err)
	}
	return data, nil
}

// InvalidateSearchCache removes all search-related cache entries (e.g. after ingestion).
func (r *RedisStore) InvalidateSearchCache(ctx context.Context) error {
	keys, err := r.client.Keys(ctx, "cinova:search:*").Result()
	if err != nil {
		return fmt.Errorf("scan search cache keys: %w", err)
	}
	if len(keys) == 0 {
		return nil
	}
	return r.client.Del(ctx, keys...).Err()
}

// ---- Recommendation cache ----

// CacheRecommendations stores recommendation results keyed by session/user ID and country.
func (r *RedisStore) CacheRecommendations(ctx context.Context, subjectID, country string, results interface{}, ttl time.Duration) error {
	key := fmt.Sprintf("cinova:rec:%s:%s", subjectID, country)
	data, err := json.Marshal(results)
	if err != nil {
		return fmt.Errorf("marshal recommendations: %w", err)
	}
	return r.client.Set(ctx, key, data, ttl).Err()
}

// GetCachedRecommendations retrieves cached recommendations. Returns (nil, nil) on miss.
func (r *RedisStore) GetCachedRecommendations(ctx context.Context, subjectID, country string) ([]byte, error) {
	key := fmt.Sprintf("cinova:rec:%s:%s", subjectID, country)
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get cached recommendations: %w", err)
	}
	return data, nil
}

// InvalidateRecommendations removes the cached recommendations for a subject.
func (r *RedisStore) InvalidateRecommendations(ctx context.Context, subjectID string) error {
	keys, err := r.client.Keys(ctx, fmt.Sprintf("cinova:rec:%s:*", subjectID)).Result()
	if err != nil {
		return fmt.Errorf("scan rec cache keys: %w", err)
	}
	if len(keys) == 0 {
		return nil
	}
	return r.client.Del(ctx, keys...).Err()
}

// ---- Rate limiting ----

// CheckRateLimit returns (allowed bool, remaining int, err). It uses a sliding
// window counter keyed by sessionID. The counter expires after rateLimitWindow.
func (r *RedisStore) CheckRateLimit(ctx context.Context, sessionID string) (bool, int, error) {
	key := fmt.Sprintf("cinova:rl:%s", sessionID)

	pipe := r.client.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, rateLimitWindow)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, 0, fmt.Errorf("rate limit pipeline: %w", err)
	}

	count := int(incr.Val())
	remaining := rateLimitMax - count
	if remaining < 0 {
		remaining = 0
	}
	return count <= rateLimitMax, remaining, nil
}

// ---- Generic key-value ----

// Get retrieves a raw string value. Returns ("", nil) on cache miss.
func (r *RedisStore) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

// Set stores a raw string value with an optional TTL.
func (r *RedisStore) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

// ---- Helpers ----

func searchCacheKey(query, country string) string {
	return fmt.Sprintf("cinova:search:%s:%s", country, query)
}

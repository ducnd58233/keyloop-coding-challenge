package configs

import "time"

const defaultCacheTTL = 60 * time.Second

// Cache is the TTL applied to cached aggregate results (A9).
type Cache struct {
	TTL time.Duration
}

func loadCache() (Cache, error) {
	ttl, err := duration("CACHE_TTL", defaultCacheTTL)
	if err != nil {
		return Cache{}, err
	}
	return Cache{TTL: ttl}, nil
}

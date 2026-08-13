package configs

import "time"

// Cache is the TTL applied to cached aggregate results (A9).
type Cache struct {
	TTL time.Duration
}

func loadCache() (Cache, error) {
	ttl, err := duration("CACHE_TTL", 60*time.Second)
	if err != nil {
		return Cache{}, err
	}
	return Cache{TTL: ttl}, nil
}

// Package configs is the only place an environment variable is read (AGENTS.md).
// cmd/api calls Load today; the mock binaries share the same defaults from T3.
package configs

import (
	"fmt"

	"github.com/joho/godotenv"
)

// Config is the process configuration, composed of one struct per concern.
type Config struct {
	HTTP     HTTP
	Sources  Sources
	Cache    Cache
	Database Database
	Log      Log
}

// Load reads .env if present, then the environment, applying the defaults
// documented in SPEC.md §6. It fails fast if the timeout ordering there is
// violated, rather than let the outer layer fire before the inner one can
// report which dependency was slow (SYSTEM_DESIGN.md DD-3).
func Load() (Config, error) {
	_ = godotenv.Load()

	httpCfg, err := loadHTTP()
	if err != nil {
		return Config{}, err
	}
	sourcesCfg, err := loadSources()
	if err != nil {
		return Config{}, err
	}
	cacheCfg, err := loadCache()
	if err != nil {
		return Config{}, err
	}
	dbCfg, err := loadDatabase()
	if err != nil {
		return Config{}, err
	}
	logCfg, err := loadLog()
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		HTTP:     httpCfg,
		Sources:  sourcesCfg,
		Cache:    cacheCfg,
		Database: dbCfg,
		Log:      logCfg,
	}

	if err := cfg.validateTimeoutOrdering(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) validateTimeoutOrdering() error {
	if c.Sources.PerSourceTimeout >= c.Sources.AggregateTimeout {
		return fmt.Errorf("PER_SOURCE_TIMEOUT (%s) must be less than AGGREGATE_TIMEOUT (%s)",
			c.Sources.PerSourceTimeout, c.Sources.AggregateTimeout)
	}
	if c.Sources.AggregateTimeout >= c.HTTP.RequestTimeout {
		return fmt.Errorf("AGGREGATE_TIMEOUT (%s) must be less than REQUEST_TIMEOUT (%s)",
			c.Sources.AggregateTimeout, c.HTTP.RequestTimeout)
	}
	return nil
}

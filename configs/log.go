package configs

// Log configures structured logging and the salt used to hash VINs before
// they reach logs, traces or the audit table (SPEC.md §8). VINHashSalt is a
// secret in production and must never be logged.
type Log struct {
	Level       string
	VINHashSalt string
}

func loadLog() (Log, error) {
	return Log{
		Level:       env("LOG_LEVEL", "info"),
		VINHashSalt: env("VIN_HASH_SALT", "dev-only-not-a-secret"),
	}, nil
}

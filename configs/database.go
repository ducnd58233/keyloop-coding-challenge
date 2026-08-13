package configs

const defaultMaxConns = 10

// Database points at the PostgreSQL instance provisioned by
// deployments/docker/docker-compose.yaml (A8).
type Database struct {
	URL      string
	MaxConns int
}

func loadDatabase() (Database, error) {
	maxConns, err := integer("DB_MAX_CONNS", defaultMaxConns)
	if err != nil {
		return Database{}, err
	}
	return Database{
		URL:      env("DATABASE_URL", "postgres://viewer:viewer@localhost:5432/viewer?sslmode=disable"),
		MaxConns: maxConns,
	}, nil
}

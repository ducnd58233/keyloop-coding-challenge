package configs

import "time"

// Sources configures the two upstream document systems (A5). Mock fault
// injection is process flags on cmd/sales and cmd/service, not environment variables.
type Sources struct {
	SalesBaseURL     string
	ServiceBaseURL   string
	PerSourceTimeout time.Duration
	AggregateTimeout time.Duration
}

func loadSources() (Sources, error) {
	perSource, err := duration("PER_SOURCE_TIMEOUT", 2*time.Second)
	if err != nil {
		return Sources{}, err
	}
	aggregate, err := duration("AGGREGATE_TIMEOUT", 2500*time.Millisecond)
	if err != nil {
		return Sources{}, err
	}

	return Sources{
		SalesBaseURL:     env("SALES_BASE_URL", "http://localhost:9100"),
		ServiceBaseURL:   env("SERVICE_BASE_URL", "http://localhost:9101"),
		PerSourceTimeout: perSource,
		AggregateTimeout: aggregate,
	}, nil
}

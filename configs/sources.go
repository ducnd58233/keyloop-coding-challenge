package configs

import "time"

// Sources configures the two upstream document systems (A5) and, since all
// three binaries share this loader, the fault injection flags read by the
// mock servers themselves (T3).
type Sources struct {
	SalesBaseURL     string
	ServiceBaseURL   string
	PerSourceTimeout time.Duration
	AggregateTimeout time.Duration
	MockLatency      time.Duration
	MockErrorRate    float64
	MockDown         bool
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
	latencyMs, err := integer("MOCK_LATENCY_MS", 0)
	if err != nil {
		return Sources{}, err
	}
	errorRate, err := float("MOCK_ERROR_RATE", 0)
	if err != nil {
		return Sources{}, err
	}
	down, err := boolean("MOCK_DOWN", false)
	if err != nil {
		return Sources{}, err
	}

	return Sources{
		SalesBaseURL:     env("SALES_BASE_URL", "http://localhost:9100"),
		ServiceBaseURL:   env("SERVICE_BASE_URL", "http://localhost:9101"),
		PerSourceTimeout: perSource,
		AggregateTimeout: aggregate,
		MockLatency:      time.Duration(latencyMs) * time.Millisecond,
		MockErrorRate:    errorRate,
		MockDown:         down,
	}, nil
}

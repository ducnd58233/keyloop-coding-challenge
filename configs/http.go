package configs

import "time"

// HTTP is the timeout the API applies to the whole request. It must exceed
// Sources.AggregateTimeout, checked in Load (SPEC.md §6).
type HTTP struct {
	Address        string
	RequestTimeout time.Duration
}

func loadHTTP() (HTTP, error) {
	requestTimeout, err := duration("REQUEST_TIMEOUT", 3*time.Second)
	if err != nil {
		return HTTP{}, err
	}
	return HTTP{
		Address:        env("HTTP_ADDR", ":8000"),
		RequestTimeout: requestTimeout,
	}, nil
}

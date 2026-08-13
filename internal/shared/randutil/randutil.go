// Package randutil uses crypto/rand so mock generation does not trip gosec G404.
package randutil

import (
	crand "crypto/rand"
	"fmt"
	"math/big"
	"time"
)

// IntN rejects n <= 0.
func IntN(n int) (int, error) {
	if n <= 0 {
		return 0, fmt.Errorf("randutil.IntN: n must be positive")
	}
	v, err := crand.Int(crand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0, fmt.Errorf("crypto/rand: %w", err)
	}
	return int(v.Int64()), nil
}

const float64Denom = 1 << 53

// Float64 is unbiased in [0, 1).
func Float64() (float64, error) {
	n, err := IntN(float64Denom)
	if err != nil {
		return 0, err
	}
	return float64(n) / float64(float64Denom), nil
}

// DurationBetween errors if high < low.
func DurationBetween(low, high time.Duration) (time.Duration, error) {
	if high < low {
		return 0, fmt.Errorf("randutil.DurationBetween: high < low")
	}
	span := int((high - low) / time.Millisecond)
	if span <= 0 {
		return low, nil
	}
	n, err := IntN(span + 1)
	if err != nil {
		return 0, err
	}
	return low + time.Duration(n)*time.Millisecond, nil
}

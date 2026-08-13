package randutil

import (
	"testing"
	"time"
)

func TestIntNRange(t *testing.T) {
	for i := 0; i < 32; i++ {
		v, err := IntN(5)
		if err != nil {
			t.Fatal(err)
		}
		if v < 0 || v >= 5 {
			t.Fatalf("IntN(5) = %d, want [0,5)", v)
		}
	}
}

func TestIntNRejectsNonPositive(t *testing.T) {
	if _, err := IntN(0); err == nil {
		t.Fatal("IntN(0) error = nil, want error")
	}
}

func TestFloat64Range(t *testing.T) {
	for i := 0; i < 32; i++ {
		v, err := Float64()
		if err != nil {
			t.Fatal(err)
		}
		if v < 0 || v >= 1 {
			t.Fatalf("Float64() = %v, want [0,1)", v)
		}
	}
}

func TestDurationBetweenRange(t *testing.T) {
	low := 100 * time.Millisecond
	high := 250 * time.Millisecond
	for i := 0; i < 32; i++ {
		d, err := DurationBetween(low, high)
		if err != nil {
			t.Fatal(err)
		}
		if d < low || d > high {
			t.Fatalf("DurationBetween = %s, want [%s,%s]", d, low, high)
		}
	}
}

func TestDurationBetweenEqual(t *testing.T) {
	d, err := DurationBetween(time.Second, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if d != time.Second {
		t.Fatalf("DurationBetween(equal) = %s, want 1s", d)
	}
}

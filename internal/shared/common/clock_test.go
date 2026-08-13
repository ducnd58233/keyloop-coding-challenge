package common

import (
	"testing"
	"time"
)

func TestFixedClockAdvanceMovesNow(t *testing.T) {
	t.Parallel()
	start := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	c := NewFixedClock(start)
	if !c.Now().Equal(start) {
		t.Fatalf("Now() = %s, want %s", c.Now(), start)
	}
	c.Advance(60 * time.Second)
	want := start.Add(60 * time.Second)
	if !c.Now().Equal(want) {
		t.Fatalf("after Advance Now() = %s, want %s", c.Now(), want)
	}
}

func TestSystemClockIsUTC(t *testing.T) {
	t.Parallel()
	now := SystemClock{}.Now()
	if now.Location() != time.UTC {
		t.Fatalf("location = %s, want UTC", now.Location())
	}
}

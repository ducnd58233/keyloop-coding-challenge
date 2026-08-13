package lifecycle

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestStartOrderThenStopReverse(t *testing.T) {
	t.Parallel()
	var order []string
	c := New(nil)
	c.Add(Hook{
		Name: "a",
		Start: func(context.Context) error {
			order = append(order, "start-a")
			return nil
		},
		Stop: func(context.Context) error {
			order = append(order, "stop-a")
			return nil
		},
	})
	c.Add(Hook{
		Name: "b",
		Start: func(context.Context) error {
			order = append(order, "start-b")
			return nil
		},
		Stop: func(context.Context) error {
			order = append(order, "stop-b")
			return nil
		},
	})
	if err := c.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	c.Stop(context.Background())
	got := strings.Join(order, ",")
	if got != "start-a,start-b,stop-b,stop-a" {
		t.Fatalf("order = %s", got)
	}
}

func TestStartFailureStopsEarlierHooks(t *testing.T) {
	t.Parallel()
	var order []string
	boom := errors.New("redis down")
	c := New(nil)
	c.Add(Hook{
		Name: "postgres",
		Start: func(context.Context) error {
			order = append(order, "start-postgres")
			return nil
		},
		Stop: func(context.Context) error {
			order = append(order, "stop-postgres")
			return nil
		},
	})
	c.Add(Hook{
		Name: "redis",
		Start: func(context.Context) error {
			order = append(order, "start-redis")
			return boom
		},
		Stop: func(context.Context) error {
			order = append(order, "stop-redis")
			return nil
		},
	})
	err := c.Start(context.Background())
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want redis down", err)
	}
	got := strings.Join(order, ",")
	if got != "start-postgres,start-redis,stop-postgres" {
		t.Fatalf("order = %s, want postgres stopped and redis not stopped", got)
	}
}

func TestEmptyChainAndNilHooks(t *testing.T) {
	t.Parallel()
	c := New(nil)
	if err := c.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	c.Stop(context.Background())

	c2 := New(nil)
	c2.Add(Hook{Name: "logger"})
	if err := c2.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	c2.Stop(context.Background())
}

func TestStopIsIdempotent(t *testing.T) {
	t.Parallel()
	n := 0
	c := New(nil)
	c.Add(Hook{
		Name:  "postgres",
		Start: func(context.Context) error { return nil },
		Stop: func(context.Context) error {
			n++
			return nil
		},
	})
	if err := c.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	c.Stop(context.Background())
	c.Stop(context.Background())
	if n != 1 {
		t.Fatalf("stop calls = %d, want 1", n)
	}
}

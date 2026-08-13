package circuitbreaker

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/ducnd58233/unified-document-viewer/internal/shared/common"
)

func TestOpensAfterThresholdThenFailFast(t *testing.T) {
	t.Parallel()
	clk := common.NewFixedClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	b := New(Settings{Name: "sales", Threshold: 3, Cooldown: time.Minute, Clock: clk})
	boom := errors.New("upstream fetch failed")
	for i := 0; i < 3; i++ {
		if err := b.Execute(func() error { return boom }); !errors.Is(err, boom) {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	calls := 0
	err := b.Execute(func() error {
		calls++
		return nil
	})
	if !errors.Is(err, ErrOpen) {
		t.Fatalf("err = %v, want ErrOpen", err)
	}
	if calls != 0 {
		t.Fatal("open breaker must not call fn")
	}
}

func TestHalfOpenProbeSuccessCloses(t *testing.T) {
	t.Parallel()
	clk := common.NewFixedClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	b := New(Settings{Name: "service", Threshold: 2, Cooldown: time.Second, Clock: clk})
	_ = b.Execute(func() error { return errors.New("x") })
	_ = b.Execute(func() error { return errors.New("x") })
	clk.Advance(time.Second)
	if err := b.Execute(func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := b.Execute(func() error { return nil }); err != nil {
		t.Fatalf("closed breaker: %v", err)
	}
}

func TestHalfOpenProbeFailureReopens(t *testing.T) {
	t.Parallel()
	clk := common.NewFixedClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	b := New(Settings{Name: "sales", Threshold: 1, Cooldown: time.Second, Clock: clk})
	_ = b.Execute(func() error { return errors.New("x") })
	clk.Advance(time.Second)
	boom := errors.New("still down")
	if err := b.Execute(func() error { return boom }); !errors.Is(err, boom) {
		t.Fatalf("probe err = %v", err)
	}
	calls := 0
	if err := b.Execute(func() error { calls++; return nil }); !errors.Is(err, ErrOpen) {
		t.Fatalf("err = %v, want ErrOpen", err)
	}
	if calls != 0 {
		t.Fatal("reopened breaker called fn")
	}
}

func TestCanceledDoesNotTrip(t *testing.T) {
	t.Parallel()
	b := New(Settings{Name: "sales", Threshold: 1, Cooldown: time.Minute})
	if err := b.Execute(func() error { return context.Canceled }); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
	calls := 0
	if err := b.Execute(func() error { calls++; return nil }); err != nil || calls != 1 {
		t.Fatalf("cancel must not open: err=%v calls=%d", err, calls)
	}
}

func TestIndependentBreakers(t *testing.T) {
	t.Parallel()
	sales := New(Settings{Name: "sales", Threshold: 1, Cooldown: time.Minute})
	service := New(Settings{Name: "service", Threshold: 1, Cooldown: time.Minute})
	_ = sales.Execute(func() error { return errors.New("sales down") })
	if err := sales.Execute(func() error { return nil }); !errors.Is(err, ErrOpen) {
		t.Fatalf("sales err = %v", err)
	}
	if err := service.Execute(func() error { return nil }); err != nil {
		t.Fatalf("service must stay closed: %v", err)
	}
}

func TestStateLogsOmitVIN(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	b := New(Settings{Name: "sales", Threshold: 1, Cooldown: time.Minute, Log: log})
	_ = b.Execute(func() error { return errors.New("upstream fetch failed") })
	out := buf.String()
	if !strings.Contains(out, "sales") || !strings.Contains(out, "open") {
		t.Fatalf("log = %q, want breaker name and open", out)
	}
	if strings.Contains(out, "1HGCM82633") || strings.Contains(strings.ToLower(out), "localhost") {
		t.Fatalf("log leaked vin or host: %s", out)
	}
}

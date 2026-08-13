// Package postgres holds the process pool and unit of work (R4, A8).
package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const connectTimeout = 8 * time.Second

// ErrNoRows lets adapters treat a miss without importing pgx.
var ErrNoRows = pgx.ErrNoRows

// Pool is shared. Repositories must not open their own connections.
type Pool struct {
	inner *pgxpool.Pool
}

// Querier is the pool or the in-flight Unit of Work transaction.
type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type txKey struct{}

// Open never puts the database URL (password) or driver text into the returned error.
func Open(ctx context.Context, url string, maxConns int) (*Pool, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, connectTimeout)
		defer cancel()
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, errors.New("postgres: invalid database url")
	}
	if maxConns > 0 {
		if maxConns > math.MaxInt32 {
			return nil, fmt.Errorf("postgres: DB_MAX_CONNS %d is too large", maxConns)
		}
		cfg.MaxConns = int32(maxConns)
	}
	p, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, errors.New("postgres: connect failed")
	}
	if err := p.Ping(ctx); err != nil {
		p.Close()
		return nil, errors.New("postgres: ping failed")
	}
	return &Pool{inner: p}, nil
}

// Close is safe on a nil pool.
func (p *Pool) Close() {
	if p == nil || p.inner == nil {
		return
	}
	p.inner.Close()
}

// Ping errors on a nil pool instead of panicking.
func (p *Pool) Ping(ctx context.Context) error {
	if p == nil || p.inner == nil {
		return errors.New("postgres: pool closed")
	}
	if err := p.inner.Ping(ctx); err != nil {
		return errors.New("postgres: ping failed")
	}
	return nil
}

func withTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// QuerierFrom returns the in-flight transaction when UnitOfWork.Within is active.
func QuerierFrom(ctx context.Context, p *Pool) Querier {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return p.inner
}

package postgres

import (
	"context"
	"errors"
)

// UnitOfWork is the only Begin/Commit/Rollback site (R4).
type UnitOfWork struct {
	pool *Pool
}

// NewUnitOfWork shares the process pool; it does not open a second connection (R4).
func NewUnitOfWork(pool *Pool) *UnitOfWork {
	return &UnitOfWork{pool: pool}
}

// Within runs fn in one transaction. Repositories join via QuerierFrom, never Begin.
func (u *UnitOfWork) Within(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := u.pool.inner.Begin(ctx)
	if err != nil {
		return errors.New("postgres: begin failed")
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()

	if err := fn(withTx(ctx, tx)); err != nil {
		return err
	}
	if err := tx.Commit(context.WithoutCancel(ctx)); err != nil {
		return errors.New("postgres: commit failed")
	}
	return nil
}

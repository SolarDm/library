package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var _ Transactor = (*transactorImpl)(nil)

type transactorImpl struct {
	logger *zap.Logger
	db     *pgxpool.Pool
}

func NewTransactor(db *pgxpool.Pool, logger *zap.Logger) *transactorImpl {
	return &transactorImpl{
		db:     db,
		logger: logger,
	}
}

func (t *transactorImpl) WithTx(ctx context.Context, f func(ctx context.Context) error) (txErr error) {
	ctxWithTx, tx, err := injectTx(ctx, t.db)

	if err != nil {
		t.logger.Error("Error while injecting transaction.", zap.Error(err))
		return fmt.Errorf("can not inject transaction, error: %w", err)
	}

	defer func() {
		if txErr != nil {
			err = tx.Rollback(ctxWithTx)
			t.logger.Error("Error while doing rollback.", zap.Error(err))
			return
		}

		txErr = tx.Commit(ctxWithTx)
		if err != nil {
			t.logger.Error("Error while commiting transaction.", zap.Error(err))
		}
	}()

	err = f(ctxWithTx)

	if err != nil {
		t.logger.Error("Error while executing function.", zap.Error(err))
		return fmt.Errorf("function execution error: %w", err)
	}

	return nil
}

func injectTx(ctx context.Context, pool *pgxpool.Pool) (context.Context, pgx.Tx, error) {
	if tx, err := extractTx(ctx); err == nil {
		return ctx, tx, nil
	}

	tx, err := pool.Begin(ctx)

	if err != nil {
		return nil, nil, err
	}

	return context.WithValue(ctx, txInjector{}, tx), tx, nil
}

type txInjector struct{}

var ErrTxNotFound = errors.New("transaction is not found in context")

func extractTx(ctx context.Context) (pgx.Tx, error) {
	tx, ok := ctx.Value(txInjector{}).(pgx.Tx)

	if !ok {
		return nil, ErrTxNotFound
	}

	return tx, nil
}

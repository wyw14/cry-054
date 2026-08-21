package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wyw14/cry-054/internal/application"
	"github.com/wyw14/cry-054/internal/domain"
)

type pgxQuerier interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type PgxStore struct {
	pool *pgxpool.Pool
	pgxRepositories
}

type pgxRepositories struct {
	q pgxQuerier
}

func OpenPostgres(ctx context.Context, databaseURL string) (*PgxStore, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}
	config.MaxConns = 12
	config.MinConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("open postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	store := &PgxStore{pool: pool}
	store.pgxRepositories = pgxRepositories{q: pool}
	return store, nil
}

func (s *PgxStore) Close() {
	if s != nil && s.pool != nil {
		s.pool.Close()
	}
}

func (s *PgxStore) WithinTransaction(ctx context.Context, operation func(context.Context, application.Repositories) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	repositories := &pgxRepositories{q: tx}
	if err := operation(ctx, repositories); err != nil {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return errors.Join(err, fmt.Errorf("rollback transaction: %w", rollbackErr))
		}
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return mapPgxError(err)
	}
	return nil
}

func mapPgxError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) {
		switch postgresError.Code {
		case "23505":
			return fmt.Errorf("unique constraint %s: %w", postgresError.ConstraintName, domain.ErrConflict)
		case "23503":
			return fmt.Errorf("foreign key %s: %w", postgresError.ConstraintName, domain.ErrInvalidInput)
		case "40001", "40P01":
			return fmt.Errorf("transaction serialization: %w", domain.ErrConflict)
		}
	}
	return err
}

func parseDatabaseMoney(raw string) (domain.Money, error) {
	money, err := domain.ParseMoney(raw)
	if err != nil {
		return domain.Money{}, fmt.Errorf("database money %q: %w", raw, err)
	}
	return money, nil
}

var _ application.Repositories = (*PgxStore)(nil)
var _ application.UnitOfWork = (*PgxStore)(nil)

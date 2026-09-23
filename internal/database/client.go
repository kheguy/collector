package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kheguy/collector/internal/repository"
	"github.com/kheguy/collector/migrations"
)

type Client struct {
	pool    pool
	storage *repository.DBStorage
}

func NewClient(ctx context.Context, dsn string) (*Client, error) {
	return newClient(
		ctx,
		dsn,
		migrations.Up,
		func(ctx context.Context, dsn string) (pool, error) {
			return pgxpool.New(ctx, dsn)
		},
	)
}

type pool interface {
	repository.Pool
	Ping(context.Context) error
	Close()
}

type migrateFunc func(string) error

type poolFactory func(context.Context, string) (pool, error)

func newClient(ctx context.Context, dsn string, migrate migrateFunc, newPool poolFactory) (*Client, error) {
	if err := migrate(dsn); err != nil {
		return nil, fmt.Errorf("apply database migrations error: %w", err)
	}

	pool, err := newPool(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("create database pool error: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database error: %w", err)
	}

	return &Client{
		pool:    pool,
		storage: repository.NewDBStorage(pool),
	}, nil
}

func (c *Client) Storage() *repository.DBStorage {
	return c.storage
}

func (c *Client) Ping(ctx context.Context) error {
	return c.pool.Ping(ctx)
}

func (c *Client) Close() {
	c.pool.Close()
}

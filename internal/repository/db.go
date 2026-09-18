package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	models "github.com/kheguy/collector/internal/model"
)

type commandExecutor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type Pool interface {
	commandExecutor
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Begin(context.Context) (pgx.Tx, error)
}

type DBStorage struct {
	pool Pool
}

func NewDBStorage(p Pool) *DBStorage {
	return &DBStorage{
		pool: p,
	}
}

func (s *DBStorage) Get(name string, ctx context.Context) (interface{}, error) {
	var mType string
	var delta pgtype.Int8
	var value pgtype.Float8

	err := s.pool.QueryRow(
		ctx,
		"SELECT mtype, delta, value FROM metrics WHERE name = $1",
		name,
	).Scan(&mType, &delta, &value)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return metricValue(mType, delta, value)
}

func (s *DBStorage) Set(name string, mType string, value interface{}, ctx context.Context) error {
	return upsertMetric(ctx, s.pool, name, mType, value)
}

func (s *DBStorage) SetBatch(metrics []models.Metrics, ctx context.Context) error {
	if len(metrics) == 0 {
		return nil
	}

	values := make([]interface{}, len(metrics))
	for i, metric := range metrics {
		value, err := batchMetricValue(metric)
		if err != nil {
			return err
		}
		values[i] = value
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	for i, metric := range metrics {
		if err := upsertMetric(ctx, tx, metric.ID, metric.MType, values[i]); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func upsertMetric(ctx context.Context, executor commandExecutor, name string, mType string, value interface{}) error {
	switch mType {
	case models.Gauge:
		gauge, ok := value.(float64)
		if !ok {
			return errors.New("invalid gauge value type")
		}

		_, err := executor.Exec(
			ctx,
			`INSERT INTO metrics (name, mtype, delta, value)
			 VALUES ($1, $2, NULL, $3)
			 ON CONFLICT (name) DO UPDATE
			 SET mtype = EXCLUDED.mtype,
			     delta = NULL,
			     value = EXCLUDED.value`,
			name,
			mType,
			gauge,
		)
		if err != nil {
			return err
		}
		return nil

	case models.Counter:
		counter, ok := value.(int)
		if !ok {
			return errors.New("invalid counter value type")
		}

		_, err := executor.Exec(
			ctx,
			`INSERT INTO metrics (name, mtype, delta, value)
			 VALUES ($1, $2, $3, NULL)
			 ON CONFLICT (name) DO UPDATE
			 SET mtype = EXCLUDED.mtype,
			     delta = COALESCE(metrics.delta, 0) + EXCLUDED.delta,
			     value = NULL`,
			name,
			mType,
			int64(counter),
		)
		if err != nil {
			return err
		}
		return nil

	default:
		return errors.New("unknown metric type")
	}
}

func (s *DBStorage) GetAll(ctx context.Context) (map[string]interface{}, error) {
	rows, err := s.pool.Query(ctx, "SELECT name, mtype, delta, value FROM metrics")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]interface{})
	for rows.Next() {
		var name string
		var mType string
		var delta pgtype.Int8
		var value pgtype.Float8

		if err := rows.Scan(&name, &mType, &delta, &value); err != nil {
			return nil, err
		}

		metric, err := metricValue(mType, delta, value)
		if err != nil {
			return nil, err
		}
		result[name] = metric
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *DBStorage) SetAll(data map[string]interface{}, ctx context.Context) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, "DELETE FROM metrics"); err != nil {
		return err
	}

	for name, value := range data {
		if err := insertMetric(ctx, tx, name, value); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *DBStorage) Clear(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, "DELETE FROM metrics"); err != nil {
		return err
	}
	return nil
}

func metricValue(mType string, delta pgtype.Int8, value pgtype.Float8) (interface{}, error) {
	switch mType {
	case models.Counter:
		if !delta.Valid {
			return nil, errors.New("counter has no delta")
		}

		counter := int(delta.Int64)
		if int64(counter) != delta.Int64 {
			return nil, errors.New("counter overflows int")
		}
		return counter, nil

	case models.Gauge:
		if !value.Valid {
			return nil, errors.New("gauge has no value")
		}
		return value.Float64, nil

	default:
		return nil, errors.New("unknown metric type")
	}
}

func insertMetric(ctx context.Context, executor commandExecutor, name string, value interface{}) error {
	var (
		mType string
		delta any
		gauge any
	)

	switch metric := value.(type) {
	case int:
		mType = models.Counter
		delta = int64(metric)
	case float64:
		mType = models.Gauge
		gauge = metric
	default:
		return errors.New("unsupported metric value type")
	}

	_, err := executor.Exec(
		ctx,
		"INSERT INTO metrics (name, mtype, delta, value) VALUES ($1, $2, $3, $4)",
		name,
		mType,
		delta,
		gauge,
	)
	if err != nil {
		return err
	}

	return nil
}

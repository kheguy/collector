package migrations

import (
	"errors"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func Up(databaseDSN string) error {
	m, err := migrate.New("file://migrations", databaseDSN)
	if err != nil {
		return err
	}
	defer func() {
		_, _ = m.Close()
	}()

	err = m.Up()
	if errors.Is(err, migrate.ErrNoChange) {
		return nil
	}
	return err
}

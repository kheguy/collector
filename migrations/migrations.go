package migrations

import (
	"embed"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed *.sql
var files embed.FS

func Up(databaseDSN string) error {
	source, err := iofs.New(files, ".")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, databaseDSN)
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

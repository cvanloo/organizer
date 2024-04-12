package organizer

import (
	"errors"
	"fmt"
	"time"
	"log"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

type SqlConnection struct {
	Driver string
	User string
	Password string
	SocketPath string
	Database string
	MaxConns int
	MaxLifetime time.Duration
}

func (s SqlConnection) String() string {
	// [username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]
	// username:password@unix(socketPath)/dbname?charset=utf8
	connStr := fmt.Sprintf(
		//"%s:%s@unix(%s)/%s?charset=utf8",
		"%s@unix(%s)/%s?charset=utf8",
		s.User,
		//s.Password,
		s.SocketPath,
		s.Database,
	)
	return connStr
}

func DB(cfg SqlConnection) (*sql.DB, error) {
	db, err := sql.Open(cfg.Driver, cfg.String())
	if err != nil {
		return nil, err
	}
	db.SetConnMaxLifetime(cfg.MaxLifetime)
	db.SetMaxOpenConns(cfg.MaxConns)
	db.SetMaxIdleConns(cfg.MaxConns)

	if err := migrate(db); err != nil {
		return db, err
	}
	return db, nil
}

var migrations = []func(*sql.Tx) error {
	m01_initial,
}

var maxVersion = int64(len(migrations))

func migrate(db *sql.DB) error {
	targetVersion := maxVersion
	var currentVersion int64

	row := db.QueryRow("select version from migrations order by version desc limit 1;")
	if err := row.Scan(&currentVersion); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to read migration table: %w", err)
	}

	if currentVersion < targetVersion {
		log.Printf("migrating database from version %d to %d", currentVersion, targetVersion)
	}

	for version := currentVersion; version < targetVersion; version++ {
		migrateFunc := migrations[version]
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("migration %d: failed to start transaction: %w", version, err)
		}
		if err = migrateFunc(tx); err != nil {
			err = fmt.Errorf("migration %d: failed to migrate: %w", version, err)
			rerr := tx.Rollback()
			return errors.Join(err, rerr)
		}
		if _, err := tx.Exec("insert into migrations () values ();"); err != nil {
			err = fmt.Errorf("migration %d: failed to bump version: %w", version, err)
			rerr := tx.Rollback()
			return errors.Join(err, rerr)
		}
		if err = tx.Commit(); err != nil {
			log.Printf("migration %d: failed to commit changes", version)
			return err
		}
	}
	log.Printf("database (v%d) up to date", targetVersion)
	return nil
}

func m01_initial(tx *sql.Tx) error {
	steps := []string{
		`create table if not exists users (
			id int primary key auto_increment
		);`,
		`create table if not exists events (
			id int primary key auto_increment
		);`,
	}
	for _, sql := range steps {
		_, err := tx.Exec(sql)
		if err != nil {
			return err
		}
	}
	return nil
}

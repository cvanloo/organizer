package organizer

import (
	"errors"
	"fmt"
	"time"
	"log"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

type (
	Repository interface {
		Prepare(db *sql.DB) error
		User(email string) (User, error)
	}
	User struct {
		ID int
		Name string
		Display string
		Email string
		Icon string
	}
)

type SqlConnection struct {
	Driver string
	User string
	Password string
	SocketPath string
	Database string
	MaxConns int
	MaxLifetime time.Duration
	UseSocket bool
}

func (s SqlConnection) String() string {
	// [username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]
	// username:password@unix(socketPath)/dbname?charset=utf8
	var connStr string
	if s.UseSocket {
		connStr = fmt.Sprintf(
			"%s@unix(%s)/%s?charset=utf8",
			s.User,
			s.SocketPath,
			s.Database,
		)
	} else {
		connStr = fmt.Sprintf(
			"%s:%s@/%s?charset=utf8",
			s.User,
			s.Password,
			s.Database,
		)
	}
	return connStr
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
	log.Printf("database v%d up to date", targetVersion)
	return nil
}

func runSteps(tx *sql.Tx, steps []string) error {
	for i, sql := range steps {
		_, err := tx.Exec(sql)
		if err != nil {
			return fmt.Errorf("sql step %d failed: %w", i, err)
		}
	}
	return nil
}

func m01_initial(tx *sql.Tx) error {
	steps := []string{
		`create table if not exists users (
			id int primary key auto_increment,
			name varchar(30) not null,
			display varchar(30) not null,
			email varchar(255) not null unique,
			icon varchar(255) default null,
			created_at datetime not null default current_timestamp,
			changed_at datetime default null,
			deleted_at datetime default null
		);`,
		`create table if not exists events (
			id int primary key auto_increment,
			created_by int not null references users (id),
			title varchar(255) not null,
			description varchar(4096) not null,
			repeats_every int not null default 0,
			repeats_scale enum ('never', 'day', 'month', 'year') not null default 'never',
			created_at datetime not null default current_timestamp,
			changed_at datetime default null,
			deleted_at datetime default null
		);`,
		`create table if not exists event_subscriptions (
			id int primary key auto_increment,
			user_id int not null references users (id),
			event_id int not null references events (id),
			unique index (user_id, event_id),
			created_at datetime not null default current_timestamp,
			changed_at datetime default null,
			deleted_at datetime default null
		);`,
	}
	return runSteps(tx, steps)
}

func m02_email_recovery(tx *sql.Tx) error {
	// @todo: find that one blog post again
	steps := []string{
		`create table if not exists email_changes (
			id int primary key auto_increment,
			email varchar(255) not null unique,
			confirm_token text(16),
			changed_from_id int references email_changes (id) default null,
			undo_token text(16) default null,
		);`,
		`alter table users change email email_id int not null references email_changes (id);`,
	}
	return runSteps(tx, steps)
}

func m03_trust(tx *sql.Tx) error {
	steps := []string{}
	return runSteps(tx, steps)
}

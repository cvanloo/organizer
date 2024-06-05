package organizer

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"time"
)

type (
	Repository interface {
		Prepare(db *sql.DB) error
		User(id UserID) (User, error)
		UserByEmail(email string) (User, error)
		Event(id EventID) (Event, error)
		CreateEvent(event Event) (Event, error)
		RegisterEvent(reg EventRegistration) (EventRegistration, error)
		DeregisterEvent(id EventRegistrationID) error
		Events() ([]Event, error)
		EventRegistration(id EventRegistrationID) (EventRegistration, error)
		EventRegistrations(eventID EventID) ([]EventRegistration, error)
	}
	UserID int
	User   struct {
		ID      UserID
		Name    string
		Display sql.NullString
		Email   string
		Icon    sql.NullString
	}
	EventID int
	Event struct {
		ID EventID
		CreatedBy UserID
		Title, Description string
		RepeatsEvery int
		RepeatsScale TimeScale
		MinParticipants sql.NullInt64
		MaxParticipants sql.NullInt64
		NumberOfParticipants int
	}
	EventRegistrationID int
	EventRegistration struct {
		ID EventRegistrationID
		User UserID
		Event EventID
		Message sql.NullString
	}
	TimeScale string
)

const (
	RepeatsNever TimeScale = "never"
	RepeatsDaily TimeScale = "daily"
	RepeatsWeekly TimeScale = "weekly"
	RepeatsMonthly TimeScale = "monthly"
	RepeatsYearly TimeScale = "yearly"
)

func (t *TimeScale) Scan(src any) error {
	*t = TimeScale(src.([]byte))
	return nil
}

func ValidScale(scale string) (TimeScale, bool) {
	switch TimeScale(scale) {
	case RepeatsNever:
		return RepeatsNever, true
	case RepeatsDaily:
		return RepeatsDaily, true
	case RepeatsWeekly:
		return RepeatsWeekly, true
	case RepeatsMonthly:
		return RepeatsMonthly, true
	case RepeatsYearly:
		return RepeatsYearly, true
	default:
		return RepeatsNever, false
	}
}

type SqlConnection struct {
	Driver      string
	User        string
	Password    string
	SocketPath  string
	Database    string
	MaxConns    int
	MaxLifetime time.Duration
	UseSocket   bool
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

func NewEvent(by UserID, title, desc string, every int, scale TimeScale, minPart, maxPart int) Event {
	return Event{
		CreatedBy: by,
		Title: title,
		Description: desc,
		RepeatsEvery: every,
		RepeatsScale: scale,
		MinParticipants: sql.NullInt64{
			Int64: int64(minPart),
			Valid: minPart != 0,
		},
		MaxParticipants: sql.NullInt64{
			Int64: int64(maxPart),
			Valid: maxPart != 0,
		},
	}
}

func NewEventRegistration(by UserID, to EventID, msg string) (reg EventRegistration) {
	reg.User = by
	reg.Event = to
	reg.Message = sql.NullString{
		String: msg,
		Valid: len(msg) > 0,
	}
	return reg
}

var migrations = []func(*sql.Tx) error{
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
			repeats_scale enum ('never', 'daily', 'weekly', 'monthly', 'yearly') not null default 'never',
			min_part_num int default null,
			max_part_num int default null,
			created_at datetime not null default current_timestamp,
			changed_at datetime default null,
			deleted_at datetime default null
		);`,
		`create table if not exists event_subscriptions (
			id int primary key auto_increment,
			user_id int not null references users (id),
			event_id int not null references events (id),
			unique index (user_id, event_id),
			message varchar(512) default null,
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
			changed_from_id int references email_changes (id),
			changed_to_id int references email_changes (id) default null,
			confirm_token text(50),
			undo_token text(50),
		);`,
		//`insert into email_changes (email, changed_from_id, confirm_token, undo_token)
		//select users.email, null, null, null from users;`,
		`alter table users change email email_id int not null references email_changes (id);`,
	}
	return runSteps(tx, steps)
}

func m03_trust(tx *sql.Tx) error {
	steps := []string{}
	return runSteps(tx, steps)
}

package organizer

import (
	"database/sql"
)

type MariaDB struct {
	db       *sql.DB
	StmtUser *sql.Stmt
	StmtEvent *sql.Stmt
	StmtCreateEvent *sql.Stmt
}

var _ Repository = (*MariaDB)(nil)

func (m *MariaDB) Prepare(db *sql.DB) error {
	m.db = db

	{
		stmt, err := db.Prepare("select id, name, display, email, icon from users where email = ? limit 1;")
		if err != nil {
			return err
		}
		m.StmtUser = stmt
	}

	{
		stmt, err := db.Prepare("select id, created_by, title, description, repeats_every, repeats_scale, min_part_num, max_part_num from events where id = ? limit 1;")
		if err != nil {
			return err
		}
		m.StmtEvent = stmt
	}

	{
		stmt, err := db.Prepare(
			`insert into events (
				created_by,
				title,
				description,
				repeats_every,
				repeats_scale,
				min_part_num,
				max_part_num
			) values (?, ?, ?, ?, ?, ?, ?);`)
		if err != nil {
			return err
		}
		m.StmtCreateEvent = stmt
	}

	return nil
}

func (m *MariaDB) User(email string) (u User, err error) {
	row := m.StmtUser.QueryRow(email)
	err = row.Scan(&u.ID, &u.Name, &u.Display, &u.Email, &u.Icon)
	return u, err
}

func (m *MariaDB) CreateEvent(event Event) (Event, error) {
	res, err := m.StmtCreateEvent.Exec(
		event.CreatedBy,
		event.Title,
		event.Description,
		event.RepeatsEvery,
		event.RepeatsScale,
		event.MinParticipants,
		event.MaxParticipants,
	)
	if err != nil {
		return event, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return event, err
	}
	event.ID = EventID(id)
	return event, nil
}

func (m *MariaDB) Event(id EventID) (e Event, err error) {
	row := m.StmtEvent.QueryRow(id)
	err = row.Scan(&e.ID, &e.CreatedBy, &e.Title, &e.Description, &e.RepeatsEvery, &e.RepeatsScale, &e.MinParticipants, &e.MaxParticipants)
	return e, err
}

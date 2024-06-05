package organizer

import (
	"database/sql"
)

type MariaDB struct {
	db       *sql.DB
	StmtUser *sql.Stmt
	StmtUserByEmail *sql.Stmt
	StmtEvent *sql.Stmt
	StmtEvents *sql.Stmt
	StmtCreateEvent *sql.Stmt
	StmtRegisterEvent *sql.Stmt
	StmtEventRegistrations *sql.Stmt
}

var _ Repository = (*MariaDB)(nil)

// @todo: how to handle rows marked as deleted?
func (m *MariaDB) Prepare(db *sql.DB) error {
	m.db = db

	{
		stmt, err := db.Prepare("select id, name, display, email, icon from users where id = ? limit 1;")
		if err != nil {
			return err
		}
		m.StmtUser = stmt
	}

	{
		stmt, err := db.Prepare("select id, name, display, email, icon from users where email = ? limit 1;")
		if err != nil {
			return err
		}
		m.StmtUserByEmail = stmt
	}

	{
		stmt, err := db.Prepare(
			`select
				events.id,
				events.created_by,
				events.title,
				events.description,
				events.repeats_every,
				events.repeats_scale,
				events.min_part_num,
				events.max_part_num,
				case when number_of_participants is NULL
					then 0
					else number_of_participants
				end as number_of_participants
			from events
			left join
				(select
					event_id,
					count(*) as number_of_participants
				from
					event_subscriptions
					group by event_id)
			as event_counts on events.id = event_counts.event_id
			where events.id = ? limit 1;`)
		if err != nil {
			return err
		}
		m.StmtEvent = stmt
	}

	{
		stmt, err := db.Prepare(
			`select
				events.id,
				events.created_by,
				events.title,
				events.description,
				events.repeats_every,
				events.repeats_scale,
				events.min_part_num,
				events.max_part_num,
				case when number_of_participants is NULL
					then 0
					else number_of_participants
				end as number_of_participants
			from events
			left join
				(select
					event_id,
					count(*) as number_of_participants
				from
					event_subscriptions
				where
					deleted_at is null
				group by event_id)
			as event_counts on events.id = event_counts.event_id;`)
		if err != nil {
			return err
		}
		m.StmtEvents = stmt
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

	{
		stmt, err := db.Prepare("insert into event_subscriptions (user_id, event_id, message) values (?, ?, ?);")
		if err != nil {
			return err
		}
		m.StmtRegisterEvent = stmt
	}

	{
		stmt, err := db.Prepare("select id, user_id, event_id, message from event_subscriptions where event_id = ?");
		if err != nil {
			return err
		}
		m.StmtEventRegistrations = stmt
	}

	return nil
}

func (m *MariaDB) User(id UserID) (u User, err error) {
	row := m.StmtUser.QueryRow(id)
	err = row.Scan(&u.ID, &u.Name, &u.Display, &u.Email, &u.Icon)
	return u, err
}

func (m *MariaDB) UserByEmail(email string) (u User, err error) {
	row := m.StmtUserByEmail.QueryRow(email)
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
	err = row.Scan(&e.ID, &e.CreatedBy, &e.Title, &e.Description, &e.RepeatsEvery, &e.RepeatsScale, &e.MinParticipants, &e.MaxParticipants, &e.NumberOfParticipants)
	return e, err
}

func (m *MariaDB) RegisterEvent(reg EventRegistration) (EventRegistration, error) {
	res, err := m.StmtRegisterEvent.Exec(reg.User, reg.Event, reg.Message)
	if err != nil {
		return reg, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return reg, err
	}
	reg.ID = EventRegistrationID(id)
	return reg, nil
}

func (m *MariaDB) EventRegistrations(eventID EventID) ([]EventRegistration, error) {
	rows, err := m.StmtEventRegistrations.Query(eventID)
	if err != nil {
		return nil, err
	}
	regs := []EventRegistration{}
	for rows.Next() {
		reg := EventRegistration{}
		if err := rows.Scan(&reg.ID, &reg.User, &reg.Event, &reg.Message); err != nil {
			return regs, err
		}
		regs = append(regs, reg)
	}
	return regs, nil
}

func (m *MariaDB) Events() ([]Event, error) {
	rows, err := m.StmtEvents.Query()
	if err != nil {
		return nil, err
	}
	events := []Event{}
	for rows.Next() {
		e := Event{}
		err := rows.Scan(&e.ID, &e.CreatedBy, &e.Title, &e.Description, &e.RepeatsEvery, &e.RepeatsScale, &e.MinParticipants, &e.MaxParticipants, &e.NumberOfParticipants)
		if err != nil {
			return events, err
		}
		events = append(events, e)
	}
	return events, nil
}

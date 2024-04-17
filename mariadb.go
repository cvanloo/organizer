package organizer

import (
	"database/sql"
)

type MariaDB struct {
	db       *sql.DB
	StmtUser *sql.Stmt
}

func (m *MariaDB) Prepare(db *sql.DB) error {
	m.db = db

	{
		stmt, err := db.Prepare("select id, name, display, email, icon from users where email = ? limit 1;")
		if err != nil {
			return err
		}
		m.StmtUser = stmt
	}

	return nil
}

var _ Repository = (*MariaDB)(nil)

func (m *MariaDB) User(email string) (u User, err error) {
	row := m.StmtUser.QueryRow(email)
	err = row.Scan(&u.ID, &u.Name, &u.Display, &u.Email, &u.Icon)
	return u, err
}

package organizer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"net/http"
	"time"

	"github.com/cvanloo/organizer/isdelve"
)

type (
	Service struct {
		url    Url
		mux    *http.ServeMux
		dbConn SqlConnection
		repo   Repository
		auth   *Authenticator
		mail   *Mailer
	}
	Url        string
	ServiceOpt func(*Service)
)

func NewService(opts ...ServiceOpt) (*Service, error) {
	// @todo: mandatory args
	s := &Service{
		url:  "http://localhost:8080/",
		mux:  http.DefaultServeMux,
		auth: NewAuthenticator(),
	}

	for _, opt := range opts {
		opt(s)
	}

	s.setupRoutes()

	if err := s.initializeDatabase(); err != nil {
		return nil, err
	}

	return s, nil
}

func WithUrl(url Url) ServiceOpt {
	return func(s *Service) {
		s.url = url
	}
}

func WithMux(mux *http.ServeMux) ServiceOpt {
	return func(s *Service) {
		s.mux = mux
	}
}

func WithDatabase(conn SqlConnection) ServiceOpt {
	return func(s *Service) {
		s.dbConn = conn
	}
}

func WithAuthentication(auth *Authenticator) ServiceOpt {
	return func(s *Service) {
		s.auth = auth
	}
}

func WithMailer(mail *Mailer) ServiceOpt {
	return func(s *Service) {
		s.mail = mail
	}
}

func (s *Service) TestUser(email string, sessionID string) (*Session, error) {
	if !isdelve.Enabled {
		return nil, errors.New("test user must only be used in debug mode")
	}
	user, err := s.repo.UserByEmail(email)
	if err != nil {
		return nil, err
	}
	session := s.auth.createSessionWithID(user.ID, SessionID(sessionID))
	if err != nil {
		return nil, err
	}
	login, err := session.RequestLogin()
	if err != nil {
		return nil, err
	}
	ok := session.InvalidateLogin(LoginID(login.Value))
	if !ok {
		panic("expected login to be valid")
	}
	return session, nil
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Service) setupRoutes() {
	mux := s.mux
	mux.Handle("/", homeOrNotFound{})
	mux.Handle("/home", HandlerWithError(s.routeIndex))
	mux.Handle("/login", HandlerWithError(s.login))
	mux.Handle("/auth", s.withSession(HandlerWithError(s.authenticate), false))
	mux.Handle("/events", s.withAuth(HandlerWithError(s.events)))
	mux.Handle("/create", s.withAuth(HandlerWithError(s.create)))
	mux.Handle("/event/", s.withAuth(HandlerWithError(s.event)))
	mux.Handle("/event/register", s.withAuth(HandlerWithError(s.eventRegister)))
	mux.Handle("/styles.css", styles)
	mux.Handle("/js/htmx.js", htmxScript)
}

func (s *Service) initializeDatabase() error {
	var repo Repository
	cfg := s.dbConn
	switch cfg.Driver {
	case "mysql":
		repo = &MariaDB{}
	default:
		return fmt.Errorf("no backend implemented for database: %s", cfg.Driver)
	}

	db, err := sql.Open(cfg.Driver, cfg.String())
	if err != nil {
		return fmt.Errorf("db dns: %w", err)
	}
	db.SetConnMaxLifetime(cfg.MaxLifetime)
	db.SetMaxOpenConns(cfg.MaxConns)
	db.SetMaxIdleConns(cfg.MaxConns)

	s.repo = repo

	{
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			return fmt.Errorf("db ping failed: %w", err)
		}
	}

	if err := migrate(db); err != nil {
		return fmt.Errorf("db migrations failed to run: %w", err)
	}

	// preparations must be run after migrations, because some prepared
	// statements might depend on tables from newer migrations.
	if err := repo.Prepare(db); err != nil {
		return err
	}
	return nil
}

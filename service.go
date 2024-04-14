package organizer

import (
	"fmt"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"net/http"
	"context"
	"time"
)

type Service struct {
	mux *http.ServeMux
	repo Repository
	auth *Authenticator
	mail *Mailer
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Service) InitDatabase(cfg SqlConnection) error {
	var repo Repository
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

	repo.Prepare(db)
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
	return nil
}

func (s *Service) RegisterRoutes() {
	mux := http.NewServeMux()
	s.mux = mux
	mux.Handle("/", homeOrNotFound{})
	mux.Handle("/index.html", HandlerWithError(routeIndex))
	mux.Handle("/login", HandlerWithError(s.login))
	mux.Handle("/events", HandlerWithError(events))
	mux.Handle("/create", HandlerWithError(create))
	mux.Handle("/event/", HandlerWithError(event))
	mux.Handle("/styles.css", styles)
	mux.Handle("/js/htmx.js", htmxScript)
}

func (s *Service) UseAuthentication(auth *Authenticator) {
	s.auth = auth
}

func (s *Service) UseMailer(mail *Mailer) {
	s.mail = mail
}

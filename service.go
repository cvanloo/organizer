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
	db *sql.DB
	mux *http.ServeMux
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Service) InitDatabase(cfg SqlConnection) error {
	db, err := sql.Open(cfg.Driver, cfg.String())
	if err != nil {
		return fmt.Errorf("db dns: %w", err)
	}
	db.SetConnMaxLifetime(cfg.MaxLifetime)
	db.SetMaxOpenConns(cfg.MaxConns)
	db.SetMaxIdleConns(cfg.MaxConns)
	s.db = db

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
	s.mux = http.NewServeMux()
	registerRoutes(s.mux)
}

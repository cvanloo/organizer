package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/cvanloo/organizer"
)

// @todo: read form config
var cfg = organizer.SqlConnection{
	Driver: "mysql",
	User: "organizer",
	//Password: "todo",
	SocketPath: "/run/mysqld/mysqld.sock",
	Database: "organizer",
	MaxConns: 50,
	MaxLifetime: 3*time.Minute,
	UseSocket: true,
}

// sudo fuser 8080/tcp -k
func main() {
	service := &organizer.Service{}
	if err := service.InitDatabase(cfg); err != nil {
		log.Fatal(err)
	}
	slog.Info("successfully connected to database", "driver", cfg.Driver, "conn", cfg.String())

	service.RegisterRoutes()

	srv := http.Server{
		Addr:    ":8080",
		Handler: service,
	}

	go func() {
		slog.Info("starting listener on :8080")
		err := srv.ListenAndServe /*TLS*/ ()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error(fmt.Sprintf("HTTP server error: %v", err))
		}
		slog.Info("stopped serving new connections")
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	//signal.Notify(c, os.Interrupt)
	<-c
	slog.Info("received interrupt, shutting down server")

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error(fmt.Sprintf("HTTP shutdown error: %v", err))
	}
	slog.Info("server gracefully shut down")
}

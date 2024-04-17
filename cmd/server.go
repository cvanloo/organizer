package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/cvanloo/organizer"

	// godotenv.Load(): loads .env from project root
	_ "github.com/joho/godotenv/autoload"
)

func check[T any](t T, err error) T {
	if err != nil {
		log.Fatalf("assert failed: %v", err)
	}
	return t
}

func check1(err error) {
	if err != nil {
		log.Fatalf("assert failed: %v", err)
	}
}

// @todo: read form config
var cfg = organizer.SqlConnection{
	Driver: "mysql",
	User:   "organizer",
	//Password: "todo",
	SocketPath:  "/run/mysqld/mysqld.sock",
	Database:    "organizer",
	MaxConns:    50,
	MaxLifetime: 3 * time.Minute,
	UseSocket:   true,
}

// sudo fuser 8080/tcp -k
func main() {
	checkEnv := func(key string) string {
		env, ok := os.LookupEnv(key)
		if !ok {
			log.Fatalf("env var not set: %s", key)
		}
		return env
	}

	mailCfg := organizer.MailConfig{
		Host:       checkEnv("MAIL_HOST"),
		Port:       check(strconv.Atoi(checkEnv("MAIL_PORT"))),
		Username:   checkEnv("MAIL_USER"),
		Password:   checkEnv("MAIL_PASS"),
		ThisSender: checkEnv("MAIL_USER"),
	}

	mux := http.NewServeMux()

	service, err := organizer.NewService(
		organizer.WithUrl("http://localhost:8080/"),
		organizer.WithMux(mux),
		organizer.WithDatabase(cfg),
		organizer.WithAuthentication(organizer.NewAuthenticator()),
		organizer.WithMailer(organizer.NewMailer(mailCfg)),
	)
	if err != nil {
		log.Fatalf("failed to initialize service: %v", err)
	}

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

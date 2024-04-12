package main

import (
	"fmt"
	"os"
	"log/slog"
	"os/signal"
	//"syscall"
	"net/http"
	"errors"
	"context"
	"time"

	"github.com/cvanloo/organizer"
)

// sudo fuser 8080/tcp -k
func main() {
	mux := http.NewServeMux()
	organizer.RegisterRoutes(mux)
	srv := http.Server{
		Addr: ":8080",
		Handler: mux,
	}

	go func() {
		slog.Info("starting listener on :8080")
		err := srv.ListenAndServe/*TLS*/()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error(fmt.Sprintf("HTTP server error: %v", err))
		}
		slog.Info("stopped serving new connections")
	}()

	c := make(chan os.Signal, 1)
	//signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	signal.Notify(c, os.Interrupt)
	<-c
	slog.Info("received interrupt, shutting down server")

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error(fmt.Sprintf("HTTP shutdown error: %v", err))
	}
	slog.Info("server gracefully shut down")
}

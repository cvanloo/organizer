package organizer

import (
	"fmt"
	"log/slog"
	"net/http"
)

type (
	HandlerWithError func(w http.ResponseWriter, r *http.Request) error

	ErrorResponder interface {
		RespondError(w http.ResponseWriter, r *http.Request) bool
	}

	HttpError struct {
		code int
		msg  string
	}
)

func (h HandlerWithError) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h(w, r); err != nil {
		if err, ok := err.(ErrorResponder); ok {
			if err.RespondError(w, r) {
				return
			}
		}
		slog.Error(err.Error(), "path", r.URL.EscapedPath(), "method", r.Method)
		http.Error(w, "internal server error", 500)
	}
}

type ErrNotFound struct {
	path string
}

func NotFound(r *http.Request) error {
	return ErrNotFound{
		path: r.URL.EscapedPath(),
	}
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("resource not found: %s", e.path)
}

func (e ErrNotFound) RespondError(w http.ResponseWriter, r *http.Request) bool {
	http.Error(w, e.Error(), http.StatusNotFound)
	return true
}

type ErrNotImplemented struct{}

func NotImplemented() error {
	return ErrNotImplemented{}
}

func (e ErrNotImplemented) Error() string {
	return "not implemented"
}

package organizer

import (
	"fmt"
	"log/slog"
	"net/http"
	"database/sql"
	"errors"
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

type ErrBadRequest struct {
	msg string
}

func BadRequest(msg string) error {
	return ErrBadRequest{msg}
}

func (e ErrBadRequest) Error() string {
	return fmt.Sprintf("bad request: %s", e.msg)
}

type ErrMaybe404 struct {
	err error
}

func Maybe404(err error) error {
	return ErrMaybe404{err}
}

func (e ErrMaybe404) Error() string {
	return e.err.Error()
}

func (e ErrMaybe404) Unwrap() error {
	return e.err
}

//func (e ErrMaybe404) Is(target error) bool {
//	return e.err == target
//}

func (e ErrMaybe404) Is404() bool {
	return errors.Is(e.err, sql.ErrNoRows)
	//return e.Is(sql.ErrNoRows)
}

func (e ErrMaybe404) RespondError(w http.ResponseWriter, r *http.Request) bool {
	if e.Is404() {
		http.Error(w, "resource not found", http.StatusNotFound)
		return true
	}
	return false
}

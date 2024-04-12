package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"log/slog"
	"html/template"
)

func main() {
	//http.Handle("/", HandlerWithError(routeNotFound))
	//http.Handle("/", redirect("/index.html"))
	http.Handle("/", homeOrNotFound{})
	http.Handle("/index.html", HandlerWithError(routeIndex))
	http.Handle("/login", HandlerWithError(login))
	http.Handle("/events", HandlerWithError(events))
	http.Handle("/create", HandlerWithError(create))
	slog.Info("starting listener on :8080")
	http.ListenAndServe/*TLS*/(":8000", nil)
}

type homeOrNotFound struct{}

func (h homeOrNotFound) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		redirect("/index.html")(w, r)
	} else {
		HandlerWithError(routeNotFound).ServeHTTP(w, r)
	}
}

func routeNotFound(w http.ResponseWriter, r *http.Request) error {
	return NotFound(r)
}

func routeIndex(w http.ResponseWriter, r *http.Request) error {
	return pages.Execute(w, "Landing", nil)
}

func redirect(to string) HandlerWithError {
	return func(w http.ResponseWriter, r *http.Request) error {
		http.Redirect(w, r, to, http.StatusMovedPermanently)
		return nil
	}
}

func login(w http.ResponseWriter, r *http.Request) error {
	return NotImplemented()
}

func events(w http.ResponseWriter, r *http.Request) error {
	events := EventListing{
		Events: []Event{
			{
				Title: "Event 1",
				Description: "Description of Event One.",
				NumberOfParticipants: 3,
			},
			{
				Title: "Event 2",
				Description: "Description of Event Two.",
				NumberOfParticipants: 5,
			},
			{
				Title: "Event 3",
				Description: "Description of Event Three.",
				NumberOfParticipants: 0,
			},
		},
	}
	return pages.Execute(w, "EventListing", events)
}

func create(w http.ResponseWriter, r *http.Request) error {
	return pages.Execute(w, "Create", nil)
}

type (
	HandlerWithError func(w http.ResponseWriter, r *http.Request) error

	ErrorResponder interface {
		RespondError(w http.ResponseWriter, r *http.Request) bool
	}

	HttpError struct {
		code int
		msg string
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

type ErrNotImplemented struct {}

func NotImplemented() error {
	return ErrNotImplemented{}
}

func (e ErrNotImplemented) Error() string {
	return "not implemented"
}

//
// Html Components
//

var (
	pages Template = Template{template.New("")}
)

func init() {
	pages.Funcs(template.FuncMap{
		"Render": Render,
	})

	template.Must(pages.Parse(HtmlLanding))
	template.Must(pages.Parse(HtmlEventListing))
	template.Must(pages.Parse(HtmlCreate))
}

type Template struct {
	*template.Template
}

func (t *Template) Execute(w io.Writer, name string, data any) error {
	return t.Template.ExecuteTemplate(w, name, data)
}

func Render(name string, data any) (template.HTML, error) {
	buf := &bytes.Buffer{}
	err := pages.Execute(buf, name, data)
	return template.HTML(buf.String()), err
}

const HtmlLanding = `
{{ define "Landing" }}
<!DOCTYPE html>
<html lang="de">
<head>
	<meta charset="utf-8">
	<title>Willkommen &mdash; Organizer</title>
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body>
	<h2>Login</h2>
	<form action="/login" method="post">
		<label for="email">Email:</label>
		<input type="email" name="email" id="email" required>
		<input type="submit" value="Anmelden">
	</form>
</body>
</html>
{{ end }}
`

type EventListing struct {
	Events []Event
}

type Event struct {
	Title, Description string
	NumberOfParticipants int
}

const HtmlEventListing = `
{{ define "EventListing" }}
<!DOCTYPE html>
<html lang="de">
<head>
	<meta charset="utf-8">
	<title>Events &mdash; Organizer</title>
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body>
	<h2>Events</h2>
{{ range .Events }}
	<div class="event-entry">
		<h3>{{ .Title }}</h3>
		<p>Teilnehmer: {{ .NumberOfParticipants }}</p>
		<p style="text-overflow: ellipsis; overflow: hidden; white-space: nowrap;">{{ .Description }}</p>
	</div>
{{ end }}
</body>
</html>
{{ end }}
`

const HtmlCreate = `
{{ define "Create" }}
<!DOCTYPE html>
<html lang="de">
<head>
	<meta charset="utf-8">
	<title>Event erstellen &mdash; Organizer</title>
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body>
	<h2>Event erstellen</h2>
	<form action="/create" method="post">
		<label for="title">Titel:</label>
		<input type="text" name="title" id="title" required>
		<label for="description">Beschreibung:</label>
		<input type="text" name="description" id="description" required>
		<input type="submit" value="Erstellen">
		<!-- Minimal number of participants -->
		<!-- Maximal number of participants -->
		<!-- Repeat automatically: (Weekly/Daily/...) -->
		<!-- maybe: Add (invite) people -->
	</form>
</body>
</html>
{{ end }}
`

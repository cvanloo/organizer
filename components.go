package organizer

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"io"
)

var (
	pages Template = Template{template.New("")}
)

func init() {
	pages.Funcs(template.FuncMap{
		"Render": Render,
	})

	template.Must(pages.Parse(HtmlLanding))
	template.Must(pages.Parse(HtmlConfirmLogin))
	template.Must(pages.Parse(HtmlEventListing))
	template.Must(pages.Parse(HtmlCreate))
	template.Must(pages.Parse(HtmlEventView))
	template.Must(pages.Parse(HtmlLoginLinkSent))
}

//go:embed htmx/htmx.js
var htmxScript StringResponder

//go:embed styles.css
var styles StringResponder

func init() {
	// @todo: uhhh...
	fileToMime[htmxScript] = "application/javascript"
	fileToMime[styles] = "text/css"
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

// @todo: add loading anim while login is being processed
const HtmlLanding = `
{{ define "Landing" }}
<!DOCTYPE html>
<html lang="de">
<head>
	<meta charset="utf-8">
	<title>Willkommen &mdash; Organizer</title>
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<link rel="stylesheet" href="/styles.css" title="Default Style">
	<script src="/js/htmx.js"></script>
</head>
<body>
	<h2>Login</h2>
	<form hx-post="/login" hx-target="body" hx-swap="innerHTML" class="list">
		<label for="email">Email:</label>
		<input type="email" name="email" id="email" required>
		<input type="submit" value="Anmelden">
	</form>
</body>
</html>
{{ end }}
`

const HtmlLoginLinkSent = `
{{ define "LoginLinkSent" }}
<main class="text-center">
	<h2>Login-Link verschickt</h2>
	<p>Überprüfe dein Postfach (auch Spam).</p>
	<p>Du kannst dieses Fenster jetzt schliessen.</p>
</main>
{{ end }}
`

type ConfirmLoginData struct {
	Token LoginID
	Csrf  string
}

const HtmlConfirmLogin = `
{{ define "ConfirmLogin" }}
<!DOCTYPE html>
<html lang="de">
<head>
	<meta charset="utf-8">
	<title>Login bestätigen &mdash; Organizer</title>
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<link rel="stylesheet" href="/styles.css" title="Default Style">
</head>
<body>
	<h2>Login Bestätigen</h2>
	<form action="/auth" method="post" class="list">
		<input type="hidden" name="token" id="token" value="{{.Token}}">
		<input type="hidden" name="csrf" id="csrf" value="{{.Csrf}}">
		<input type="submit" value="Login Bestätigen">
	</form>
</body>
</html>
{{ end }}
`

type (
	EventListing struct {
		Events []Event
	}
	TimeScale int
	EventID   int
	Event     struct {
		ID                   EventID
		Title, Description   string
		RepeatsEvery         int
		RepeatsScale         TimeScale
		NumberOfParticipants int
	}
)

const (
	RepeatsNever TimeScale = iota
	RepeatsDaily
	RepeatsWeekly
	RepeatsMonthly
	RepeatsYearly
)

func (e Event) DoesRepeat() bool {
	return e.RepeatsScale != RepeatsNever
}

func (e Event) RepeatsText() string {
	switch e.RepeatsScale {
	case RepeatsNever:
		return "wiederholt sich nie"
	case RepeatsDaily:
		if e.RepeatsEvery == 1 {
			return "wiederholt sich jeden Tag"
		} else {
			return fmt.Sprintf("wiederholt sich alle %d Tage", e.RepeatsEvery)
		}
	case RepeatsWeekly:
		if e.RepeatsEvery == 1 {
			return "wiederholt sich jede Woche"
		} else {
			return fmt.Sprintf("wiederholt sich alle %d Wochen", e.RepeatsEvery)
		}
	case RepeatsMonthly:
		if e.RepeatsEvery == 1 {
			return "wiederholt sich jeden Monat"
		} else {
			return fmt.Sprintf("wiederholt sich alle %d Monate", e.RepeatsEvery)
		}
	case RepeatsYearly:
		if e.RepeatsEvery == 1 {
			return "wiederholt sich jedes Jahr"
		} else {
			return fmt.Sprintf("wiederholt sich alle %d Jahre", e.RepeatsEvery)
		}
	}
	panic("unreachable")
}

// @todo: show event listing even when logged out (just without options requiring an account)
const HtmlEventListing = `
{{ define "EventListing" }}
<!DOCTYPE html>
<html lang="de">
<head>
	<meta charset="utf-8">
	<title>Events &mdash; Organizer</title>
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<link rel="stylesheet" href="/styles.css" title="Default Style">
	<script src="/js/htmx.js"></script>
</head>
<body>
	<h2>Events</h2>
{{ range .Events }}
	<div class="event-entry">
		<h3><a href="/event?id={{ .ID }}">{{ .Title }}</a></h3>
{{ if .DoesRepeat }}
		<p>({{ .RepeatsText }})</p>
{{ end }}
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
	<link rel="stylesheet" href="/styles.css" title="Default Style">
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

type EventDetails struct {
	Event
	Participants []Participant
	Discussion   []Comment
	Csrf         string // @todo: CsrfID (the other place(s) as well!)
}

type Participant struct {
	FullName, acceptMessage string
}

func (p Participant) AcceptMessage() string {
	if p.acceptMessage == "" {
		return "Nimmt am Event teil."
	}
	return p.acceptMessage
}

type Comment struct {
	Author, Message string
}

const HtmlEventView = `
{{ define "EventView" }}
<!DOCTYPE html>
<html lang="de">
<head>
	<meta charset="utf-8">
	<title>Eventansicht &mdash; Organizer</title>
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<link rel="stylesheet" href="/styles.css" title="Default Style">
</head>
<body>
	<div class="event-info">
		<h2>{{ .Title }}</h2>
		<p>{{ .Description }}</p>
{{ if .DoesRepeat }}
		<p>({{ .RepeatsText }})</p>
{{ end }}
		<p>Anzahl Teilnehmer: {{ .NumberOfParticipants }}</p>
	</div>
	<div class="event-register">
		<form action="/event/register" method="post">
			<input type="hidden" name="csrf" id="csrf" value="{{.Csrf}}">
			<input type="hidden" name="event" id="event" value="{{.ID}}">
			<input type="text" name="message" id="message" value="Ich mache mit!" />
			<input type="submit" value="Eintragen">
		</form>
	</div>
	<div class="event-participants">
{{ range .Participants }}
		<div class="participant">
			<p>{{ .FullName }}</p>
			<p>{{ .AcceptMessage }}</p>
		</div>
{{ end }}
	</div>
	<div class="event-discussion">
{{ range .Discussion }}
	{{ block "Comment" . }}
		<div class="comment-entry">
			<p>{{ .Author }}</p>
			<p>{{ .Message }}</p>
		</div>
	{{ end }}
{{ end }}
		<div class="comment-box">
			<form action="/comment" method="post">
				<label for="comment">Kommentar verfassen:</label>
				<input type="hidden" name="csrf" id="csrf" value="{{.Csrf}}">
				<input type="hidden" name="event" id="event" value="{{.ID}}">
				<input type="text" name="comment" id="comment" required>
				<input type="submit" value="Kommentieren">
			</form>
		</div>
	</div>
</body>
</html>
{{ end }}
`

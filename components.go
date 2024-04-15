package organizer

import (
	"bytes"
	"html/template"
	"io"
	_ "embed"
)

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

type EventListing struct {
	Events []Event
}

type Event struct {
	Title, Description   string
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
	<link rel="stylesheet" href="/styles.css" title="Default Style">
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
		<p>Anzahl Teilnehmer: {{ .NumberOfParticipants }}</p>
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
			<!-- @todo: only if logged in -->
			<!-- include event id in form data / csrf token -->
			<form action="/comment" method="post">
				<label for="comment">Kommentar verfassen:</label>
				<input type="text" name="comment" id="comment" required>
				<input type="submit" value="Kommentieren">
			</form>
		</div>
	</div>
</body>
</html>
{{ end }}
`

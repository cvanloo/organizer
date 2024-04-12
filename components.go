package organizer

import (
	"html/template"
	"io"
	"bytes"
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

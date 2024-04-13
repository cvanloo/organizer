package organizer

import (
	"net/http"
)

func registerRoutes(mux *http.ServeMux) {
	mux.Handle("/", homeOrNotFound{})
	mux.Handle("/index.html", HandlerWithError(routeIndex))
	mux.Handle("/login", HandlerWithError(login))
	mux.Handle("/events", HandlerWithError(events))
	mux.Handle("/create", HandlerWithError(create))
	mux.Handle("/event/", HandlerWithError(event))
	mux.Handle("/styles.css", styles)
	mux.Handle("/js/htmx.js", htmxScript)
}

type StringResponder string

func (s StringResponder) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if mime, ok := fileToMime[s]; ok {
		hdrs := w.Header()
		hdrs.Set("Content-Type", mime)
	}
	w.Write([]byte(s))
}

var fileToMime = map[StringResponder]string{}

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
				Title:                "Event 1",
				Description:          "Description of Event One.",
				NumberOfParticipants: 3,
			},
			{
				Title:                "Event 2",
				Description:          "Description of Event Two.",
				NumberOfParticipants: 5,
			},
			{
				Title:                "Event 3",
				Description:          "Description of Event Three.",
				NumberOfParticipants: 0,
			},
		},
	}
	return pages.Execute(w, "EventListing", events)
}

func event(w http.ResponseWriter, r *http.Request) error {
	event := EventDetails{
		Event: Event{
			Title:                "Event 1",
			Description:          "Description for Event One.",
			NumberOfParticipants: 3,
		},
		Participants: []Participant{
			{
				FullName: "Max Muster",
			},
			{
				FullName:      "Heinz Müller",
				acceptMessage: "Komme gerne.",
			},
		},
		Discussion: []Comment{
			{
				Author:  "Max Muster",
				Message: "Ich hätte da mal eine Frage...",
			},
		},
	}
	return pages.Execute(w, "EventView", event)
}

func create(w http.ResponseWriter, r *http.Request) error {
	return pages.Execute(w, "Create", nil)
}

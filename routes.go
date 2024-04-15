package organizer

import (
	"net/http"
	"time"
)

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

func (s *Service) login(w http.ResponseWriter, r *http.Request) error {
	email := r.FormValue("email")
	if email == "" {
		return BadRequest("missing field: email")
	}

	user, err := s.repo.User(email)
	if err != nil {
		return Maybe404(err)
	}

	token, err := s.auth.CreateLogin(user.ID)
	if err != nil {
		return err
	}

	err = s.mail.SendLoginLink(user.Email, TokenLink{
		Token: token.Token,
		Where: s.url,
	})
	if err != nil {
		return err
	}

	session := &http.Cookie{
		Name: "session",
		Value: "@todo",
		Expires: time.Now().Add(time.Hour*24*30), // @todo: config
		// etc. ...
	}
	http.SetCookie(w, session)

	return pages.Execute(w, "LoginLinkSent", nil)
}

func (s *Service) authenticate(w http.ResponseWriter, r *http.Request) error {
	token := r.FormValue("token")
	if token == "" {
		return BadRequest("missing parameter: token")
	}

	// @todo: implement session cookie
	sessionCookie, err := r.Cookie("session")
	if err != nil {
		return Unauthorized()
	}
	session := sessionCookie.Value

	user, err := s.auth.UserFromSession(session)
	if err != nil {
		return Unauthorized()
	}

	switch r.Method {
	case http.MethodGet:
		if !s.auth.HasValidLoginRequest(user.ID) {
			return Unauthorized()
		}
		return pages.Execute(w, "ConfirmLogin", nil /* csrf token? */)
	case http.MethodPost:
		// @todo: invalidate token
		if err := s.auth.ValidateLogin(user.ID, token); err != nil {
			return Unauthorized()
		}
		// @todo: mark session as authenticated
		w.WriteHeader(http.StatusOK)
		return nil
	default:
		return MethodNotAllowed()
	}

	return Unauthorized()
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

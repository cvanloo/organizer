package organizer

import (
	"context"
	"net/http"
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
		redirect("/home")(w, r)
	} else {
		HandlerWithError(routeNotFound).ServeHTTP(w, r)
	}
}

func routeNotFound(w http.ResponseWriter, r *http.Request) error {
	return NotFound(r)
}

func redirect(to string) HandlerWithError {
	return func(w http.ResponseWriter, r *http.Request) error {
		http.Redirect(w, r, to, http.StatusMovedPermanently)
		return nil
	}
}

func (s *Service) routeIndex(w http.ResponseWriter, r *http.Request) error {
	session, ok := s.auth.SessionFromRequest(r)
	if ok && session.Authenticated {
		http.Redirect(w, r, "/events", http.StatusFound)
		return nil
	}
	return pages.Execute(w, "Landing", nil)
}

func (s *Service) withSession(next HandlerWithError, mustBeAuthed bool) HandlerWithError {
	return func(w http.ResponseWriter, r *http.Request) error {
		session, ok := s.auth.SessionFromRequest(r)
		if !ok {
			return Unauthorized()
		}
		if mustBeAuthed && !session.Authenticated {
			return Unauthorized()
		}
		ctx := context.WithValue(r.Context(), "SESSION", session)
		return next(w, r.WithContext(ctx))
	}
}

func (s *Service) withAuth(next HandlerWithError) HandlerWithError {
	return s.withSession(next, true)
}

func (s *Service) login(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return MethodNotAllowed()
	}

	email := r.FormValue("email")
	if email == "" {
		return BadRequest("missing field: email")
	}

	user, err := s.repo.User(email)
	if err != nil {
		return Maybe404(err)
	}

	sessionToken, err := s.auth.CreateSession(user.ID)
	if err != nil {
		return err
	}

	token, err := s.auth.CreateLogin(user.ID)
	if err != nil {
		return err
	}

	err = s.mail.SendLoginLink(user.Email, TokenLink{
		Token: token.Value,
		Where: s.url,
	})
	if err != nil {
		return err
	}

	session := &http.Cookie{
		Name:    "session",
		Value:   sessionToken.Value,
		Expires: sessionToken.Expires(s.auth.sessionTokenExpiryLimit),
		// Domain: defaults to host of current document URL, not including subdomains
		HttpOnly: true, // forbids access via Document.cookie / will still be sent with JS-initiated requests
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
	}
	http.SetCookie(w, session)

	return pages.Execute(w, "LoginLinkSent", nil)
}

func (s *Service) authenticate(w http.ResponseWriter, r *http.Request) error {
	token := LoginID(r.FormValue("token"))
	if token == "" {
		return BadRequest("missing parameter: token")
	}

	session, valid := r.Context().Value("SESSION").(*SessionToken)
	if !valid {
		// Technically, should never reach this case.
		return Unauthorized()
	}

	user := session.User

	switch r.Method {
	case http.MethodGet:
		if !s.auth.HasValidLoginRequest(user) {
			return Unauthorized()
		}
		data := ConfirmLoginData{
			Token: token,
			Csrf:  "", // @todo: not implemented / necessary?
		}
		return pages.Execute(w, "ConfirmLogin", data)
	case http.MethodPost:
		if err := s.auth.ValidateLogin(user, token); err != nil {
			return Unauthorized()
		}
		session.Authenticated = true
		//w.WriteHeader(http.StatusOK)
		http.Redirect(w, r, "/", http.StatusSeeOther)
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

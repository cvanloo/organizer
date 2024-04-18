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
	if ok && session.IsAuthenticated() {
		http.Redirect(w, r, "/events", http.StatusFound)
		return nil
	}
	return pages.Execute(w, "Landing", nil)
}

func (s *Service) withSession(next HandlerWithError, mustBeAuthed bool) HandlerWithError {
	return func(w http.ResponseWriter, r *http.Request) error {
		session, ok := s.auth.SessionFromRequest(r)
		if !ok {
			//return Unauthorized()
			return redirect("/home")(w, r)
		}
		if mustBeAuthed && !session.IsAuthenticated() {
			//return Unauthorized()
			return redirect("/home")(w, r)
		}
		ctx := context.WithValue(r.Context(), "SESSION", session)
		return next(w, r.WithContext(ctx))
	}
}

func (s *Service) withAuth(next HandlerWithError) HandlerWithError {
	return s.withSession(next, true)
}

func (s *Service) login(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost { // @todo: do check for every request handler
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

	session, err := s.auth.CreateSession(user.ID)
	if err != nil {
		return err
	}

	login, err := session.RequestLogin()
	if err != nil {
		return err
	}

	if err := s.mail.SendLoginLink(user.Email, TokenLink{
		Token: login.Value,
		Where: s.url,
	}); err != nil {
		return err
	}

	sessionCookie := &http.Cookie{
		Name:    "session",
		Value:   session.Value,
		Expires: session.Expires(s.auth.sessionTokenExpiryLimit),
		// Domain: defaults to host of current document URL, not including subdomains
		HttpOnly: true, // forbids access via Document.cookie / will still be sent with JS-initiated requests
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
	}
	http.SetCookie(w, sessionCookie)

	return pages.Execute(w, "LoginLinkSent", nil)
}

func (s *Service) authenticate(w http.ResponseWriter, r *http.Request) error {
	login := LoginID(r.FormValue("token"))
	if login == "" {
		return BadRequest("missing parameter: token")
	}

	session, valid := r.Context().Value("SESSION").(*Session)
	if !valid {
		// Technically, should never reach this case.
		return Unauthorized()
	}

	switch r.Method {
	case http.MethodGet:
		if !session.HasValidLoginRequest() {
			return Unauthorized()
		}
		csrf, err := session.RequestCsrf()
		if err != nil {
			return err
		}
		data := ConfirmLoginData{
			Token: login,
			Csrf:  csrf.Value,
		}
		return pages.Execute(w, "ConfirmLogin", data)
	case http.MethodPost:
		csrf := CsrfID(r.FormValue("csrf"))
		if csrf == "" {
			return BadRequest("missing parameter: csrf")
		}
		if !session.InvalidateCsrf(csrf) {
			return Unauthorized()
		}
		if !session.InvalidateLogin(login) {
			return Unauthorized()
		}
		// @todo: for all request handlers: change response depending on requested content-type?
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

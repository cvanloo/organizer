package organizer

import (
	"context"
	"log"
	"fmt"
	"net/http"
	"strconv"
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

	user, err := s.repo.UserByEmail(email)
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

func (s *Service) logout(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return MethodNotAllowed()
	}

	session, valid := r.Context().Value("SESSION").(*Session)
	if !valid {
		// Technically, should never reach this case.
		return Unauthorized()
	}

	session.Delete()

	removeCookie := &http.Cookie{
		Name:    "session",
		Value: "gone with the wind",
		Expires: time.Time{},
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
	}
	http.SetCookie(w, removeCookie)

	hdr := w.Header()
	hdr.Set("HX-Redirect", "/")

	w.WriteHeader(http.StatusOK)
	return nil
}

func (s *Service) events(w http.ResponseWriter, r *http.Request) error {
	events, err := s.repo.Events()
	if err != nil {
		return Maybe404(err)
	}

	// @todo: turn this into a dto package function
	eventsDto := EventListing{}
	for _, event := range events {
		eventsDto.Events = append(eventsDto.Events, EventInfo{
			ID: event.ID,
			Title: event.Title,
			Description: event.Description,
			RepeatsEvery: event.RepeatsEvery,
			RepeatsScale: event.RepeatsScale,
			NumberOfParticipants: event.NumberOfParticipants,
		})
	}

	return pages.Execute(w, "EventListing", eventsDto)
}

func (s *Service) event(w http.ResponseWriter, r *http.Request) error {
	session, valid := r.Context().Value("SESSION").(*Session)
	if !valid {
		// Technically, should never reach this case.
		return Unauthorized()
	}
	csrf, err := session.RequestCsrf()
	if err != nil {
		return err
	}

	eventIDStr := r.FormValue("id")
	if eventIDStr == "" {
		return BadRequest("missing field: id")
	}
	eventID, err := strconv.Atoi(eventIDStr)
	if err != nil {
		return BadRequest("invalid value for field id: must be a number")
	}
	event, err := s.repo.Event(EventID(eventID))
	if err != nil {
		return Maybe404(err)
	}
	eventRegs, err := s.repo.EventRegistrations(event.ID)
	if err != nil {
		// @robustness: event should definitely exist here
		return Maybe404(err)
	}

	// @todo: refactor this stuff out into dto package
	parts := make([]Participant, len(eventRegs))
	userSub := EventRegistrationID(-1)
	var userParticipant Participant
	for i := range eventRegs {
		user, err := s.repo.User(eventRegs[i].User)
		if err != nil {
			// @robustness: not found here would be an internal server error though
			return Maybe404(err)
		}

		part := Participant{}
		part.DisplayName = user.Name
		if user.Display.Valid {
			part.DisplayName = user.Display.String
		}
		part.acceptMessage = ""
		if eventRegs[i].Message.Valid {
			part.acceptMessage = eventRegs[i].Message.String
		}
		parts[i] = part

		if user.ID == session.User {
			userSub = eventRegs[i].ID
			userParticipant = part
		}
	}

	eventDTO := EventDetails{
		ThisUser: session.User,
		EventInfo: *((&EventInfo{}).From(event)), // @todo: No.
		Participants: parts,
		Discussion: []Comment{}, // @todo: impl
		Csrf: csrf.Value,
		SubID: userSub,
		Participant: userParticipant,
	}
	return pages.Execute(w, "EventView", eventDTO)
}

func (s *Service) create(w http.ResponseWriter, r *http.Request) error {
	session, valid := r.Context().Value("SESSION").(*Session)
	if !valid {
		// Technically, should never reach this case.
		return Unauthorized()
	}

	switch r.Method {
	default:
		return MethodNotAllowed()
	case http.MethodGet:
		return pages.Execute(w, "Create", nil)
	case http.MethodPost:
		title := r.FormValue("title")
		desc := r.FormValue("description")
		repeats := r.FormValue("repeats") == "on"
		every := r.FormValue("every")
		scale := r.FormValue("scale")
		hasMinPart := r.FormValue("min_part") == "on"
		minPartNum := r.FormValue("min_part_num")
		hasMaxPart := r.FormValue("max_part") == "on"
		maxPartNum := r.FormValue("max_part_num")

		repeatsEvery := 0
		if repeats {
			val, err := strconv.Atoi(every)
			if err != nil {
				return BadRequest("invalid value for field every: must be a number")
			}
			repeatsEvery = val
		}

		repeatsScale := RepeatsNever
		if repeats {
			val, ok := ValidScale(scale)
			if !ok {
				return BadRequest("invalid value for field scale: must be one of never, daily, weekly, monthly, or yearly")
			}
			repeatsScale = val
		}

		minPart := 0
		if hasMinPart {
			val, err := strconv.Atoi(minPartNum)
			if err != nil {
				return BadRequest("invalid value for field min_part_num: must be a number")
			}
			minPart = val
		}

		maxPart := 0
		if hasMaxPart {
			val, err := strconv.Atoi(maxPartNum)
			if err != nil {
				return BadRequest("invalid value for field max_part_num: must be a number")
			}
			maxPart = val
		}

		event, err := s.repo.CreateEvent(NewEvent(
			session.User,
			title,
			desc,
			repeatsEvery,
			repeatsScale,
			minPart,
			maxPart,
		))
		if err != nil {
			return err
		}

		// @todo: make redirectHtmx function?
		hdr := w.Header()
		hdr.Set("HX-Redirect", fmt.Sprintf("/event?id=%d", event.ID))
		w.WriteHeader(http.StatusCreated)
		return nil
		//return redirect(fmt.Sprintf("/event?id=%d", event.ID))(w, r)
	}
}

func (s *Service) eventRegister(w http.ResponseWriter, r *http.Request) error {
	session, valid := r.Context().Value("SESSION").(*Session)
	if !valid {
		// Technically, should never reach this case.
		return Unauthorized()
	}
	csrf := CsrfID(r.FormValue("csrf"))
	if csrf == "" {
		return BadRequest("missing field: csrf")
	}
	if !session.InvalidateCsrf(csrf) {
		return Unauthorized()
	}
	event := r.FormValue("event")
	if event == "" {
		return BadRequest("missing field: event")
	}
	eventID, err := strconv.Atoi(event)
	if err != nil {
		return BadRequest("invalid value for field even: must be a number")
	}
	msg := r.FormValue("message")

	reg, err := s.repo.RegisterEvent(NewEventRegistration(
		session.User,
		EventID(eventID),
		msg,
	))
	if err != nil {
		return err
	}

	user, err := s.repo.User(reg.User)
	if err != nil {
		// @robustness: shouldn't happen that the user is not found
		log.Printf("could not find user: %v, got error: %v", reg.User, err)
		return Maybe404(err)
	}

	var participantInfo Participant
	participantInfo.DisplayName = user.Name
	if user.Display.Valid {
		participantInfo.DisplayName = user.Display.String
	}
	if reg.Message.Valid {
		participantInfo.acceptMessage = reg.Message.String
	}
	csrfToken, err := session.RequestCsrf()
	if err != nil {
		return err
	}
	deregInfo := UserDeregister{
		Participant: participantInfo,
		Csrf: csrfToken.Value,
		SubID: reg.ID,
	}
	return pages.Execute(w, "UserDeregister", deregInfo)
}

func (s *Service) eventDeregister(w http.ResponseWriter, r *http.Request) error {
	session, valid := r.Context().Value("SESSION").(*Session)
	if !valid {
		// Technically, should never reach this case.
		return Unauthorized()
	}
	csrf := CsrfID(r.FormValue("csrf"))
	if csrf == "" {
		return BadRequest("missing field: csrf")
	}
	if !session.InvalidateCsrf(csrf) {
		return Unauthorized()
	}
	subStr := r.FormValue("subscription_id")
	if subStr == "" {
		return BadRequest("missing field: subscription_id")
	}
	subNum, err := strconv.Atoi(subStr)
	if err != nil {
		return BadRequest("invalid value for field even: must be a number")
	}
	subID := EventRegistrationID(subNum)

	sub, err := s.repo.EventRegistration(subID)
	if err != nil {
		return Maybe404(err)
	}
	if sub.User != session.User {
		// don't leak that there is an id of a different user
		// @todo: while we're at it: shouldn't we be using hash ids? (https://sqids.org/?hashids)
		return NotFound(r) // @todo: maybe make it clearer in the error message that it's the id that was "not found", and not the url path
	}

	err = s.repo.DeregisterEvent(subID)
	if err != nil {
		return err
	}

	csrfToken, err := session.RequestCsrf()
	if err != nil {
		return err
	}
	regInfo := UserRegister{
		Csrf: csrfToken.Value,
		ID: sub.Event,
	}
	return pages.Execute(w, "UserRegister", regInfo)
}

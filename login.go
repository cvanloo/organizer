package organizer

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"time"
)

type (
	Token struct {
		Value   string
		Created time.Time
	}
	LoginToken struct {
		Token
	}
	LoginID      string
	SessionToken struct {
		Token
		User          UserID
		Authenticated bool
	}
	SessionID string
	CsrfToken struct {
		Token
	}
	CsrfID string
)

func NewToken(value string) Token {
	return Token{
		Value:   value,
		Created: time.Now(),
	}
}

func (t Token) HasExpired(limit time.Duration) bool {
	now := time.Now()
	return t.Expires(limit).Compare(now) <= 0
}

func (t Token) Expires(limit time.Duration) time.Time {
	return t.Created.Add(limit)
}

type (
	Authenticator struct {
		loginTokens             map[UserID]LoginToken
		sessionTokens              map[SessionID]*SessionToken
		csrfTokens              map[SessionID]CsrfToken
		tokenLength             int
		loginTokenExpiryLimit   time.Duration
		sessionTokenExpiryLimit time.Duration
		csrfTokenExpiryLimit    time.Duration
	}
	AuthOpt func(*Authenticator)
)

func NewAuthenticator(opts ...AuthOpt) *Authenticator {
	auth := &Authenticator{
		loginTokens:             map[UserID]LoginToken{},
		sessionTokens:           map[SessionID]*SessionToken{},
		csrfTokens:              map[SessionID]CsrfToken{},
		tokenLength:             50,
		loginTokenExpiryLimit:   10 * time.Minute,
		sessionTokenExpiryLimit: time.Hour * 24 * 7,
		csrfTokenExpiryLimit:    10 * time.Minute,
	}
	for _, opt := range opts {
		opt(auth)
	}
	return auth
}

func WithTokenLength(length int) AuthOpt {
	return func(a *Authenticator) {
		a.tokenLength = length
	}
}

func WithLoginTokenExpiryLimit(d time.Duration) AuthOpt {
	return func(a *Authenticator) {
		a.loginTokenExpiryLimit = d
	}
}

func WithSessionTokenExpiryLimit(d time.Duration) AuthOpt {
	return func(a *Authenticator) {
		a.sessionTokenExpiryLimit = d
	}
}

func WithCsrfTokenExpiryLimit(d time.Duration) AuthOpt {
	return func(a *Authenticator) {
		a.csrfTokenExpiryLimit = d
	}
}

func (a *Authenticator) SessionFromRequest(r *http.Request) (*SessionToken, bool) {
	sessionCookie, err := r.Cookie("session")
	if err != nil {
		return nil, false
	}
	sessionID := SessionID(sessionCookie.Value)
	session, ok := a.SessionByID(sessionID)
	return session, ok
}

func (a *Authenticator) SessionByID(session SessionID) (*SessionToken, bool) {
	t, ok := a.sessionTokens[session]
	if !ok {
		return nil, false
	}
	if t.HasExpired(a.sessionTokenExpiryLimit) {
		delete(a.sessionTokens, session)
		return nil, false
	}
	return t, true
}

func (a *Authenticator) CreateSession(u UserID) (*SessionToken, error) {
	tokenStr, err := randomToken(a.tokenLength)
	if err != nil {
		return nil, err
	}
	token := &SessionToken{
		Token:         NewToken(tokenStr),
		User:          u,
		Authenticated: false,
	}
	a.sessionTokens[SessionID(tokenStr)] = token
	return token, nil
}


// ------ @todo: interface design


func (a *Authenticator) CreateLogin(u UserID) (LoginToken, error) {
	tokenStr, err := randomToken(a.tokenLength)
	if err != nil {
		var zero LoginToken
		return zero, err
	}
	token := LoginToken{NewToken(tokenStr)}
	a.loginTokens[u] = token
	return token, nil
}

func (a *Authenticator) HasValidLoginRequest(u UserID) bool {
	t, ok := a.loginTokens[u]
	if !ok {
		return false
	}

	return !t.HasExpired(a.loginTokenExpiryLimit)
}

func (a *Authenticator) ValidateLogin(u UserID, loginID LoginID) error {
	t, ok := a.loginTokens[u]
	if !ok {
		// @todo: app/business logic errors
		return errors.New("login token expired")
	}

	if t.HasExpired(a.loginTokenExpiryLimit) {
		// @todo: app/business logic errors
		return errors.New("login token expired")
	}

	if string(loginID) != t.Value {
		// @todo: app/business logic errors
		return errors.New("login token expired")
	}

	delete(a.loginTokens, u)
	return nil
}

func (a *Authenticator) CreateCsrfForSession(session *SessionToken) (CsrfToken, error) {
	tokenStr, err := randomToken(a.tokenLength)
	if err != nil {
		return CsrfToken{}, err
	}
	token := CsrfToken{
		Token: NewToken(tokenStr),
	}
	a.csrfTokens[SessionID(session.Value)] = token
	return token, nil
}

func (a *Authenticator) ValidateCsrfForSession(session *SessionToken, csrf CsrfID) error {
	token, ok := a.csrfTokens[SessionID(session.Value)]
	if !ok {
		// @todo: app/business logic errors
		return errors.New("invalid csrf token")
	}

	if token.HasExpired(a.csrfTokenExpiryLimit) {
		// @todo: app/business logic errors
		return errors.New("invalid csrf token")
	}

	if string(csrf) != token.Value {
		// @todo: app/business logic errors
		return errors.New("invalid csrf token")
	}

	delete(a.csrfTokens, SessionID(session.Value))
	return nil
}

func randomToken(length int) (string, error) {
	bs := make([]byte, length)
	_, err := rand.Read(bs)
	return base64.URLEncoding.EncodeToString(bs), err
}

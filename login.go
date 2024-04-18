package organizer

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"
)

type (
	Token struct {
		Value   string
		Created time.Time
		Valid   bool
	}
	LoginID      string
	LoginToken struct {
		Token
	}
	CsrfID string
	CsrfToken struct {
		Token
	}
	SessionID string
	Session struct {
		Token
		User          UserID
		authenticated bool
		csrf          CsrfToken
		login         LoginToken
		auth          *Authenticator
	}
)

func NewToken(value string) Token {
	return Token{
		Value:   value,
		Created: time.Now(),
		Valid:   true,
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
		sessions           map[SessionID]*Session
		tokenLength             int
		loginTokenExpiryLimit   time.Duration
		sessionTokenExpiryLimit time.Duration
		csrfTokenExpiryLimit    time.Duration
	}
	AuthOpt func(*Authenticator)
)

func NewAuthenticator(opts ...AuthOpt) *Authenticator {
	auth := &Authenticator{
		sessions:           map[SessionID]*Session{},
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

func (a *Authenticator) SessionFromRequest(r *http.Request) (*Session, bool) {
	sessionCookie, err := r.Cookie("session")
	if err != nil {
		return nil, false
	}
	sessionID := SessionID(sessionCookie.Value)
	session, ok := a.SessionByID(sessionID)
	return session, ok
}

func (a *Authenticator) SessionByID(session SessionID) (*Session, bool) {
	t, ok := a.sessions[session]
	if !ok {
		return nil, false
	}
	if t.HasExpired(a.sessionTokenExpiryLimit) {
		delete(a.sessions, session)
		return nil, false
	}
	return t, true
}

func (a *Authenticator) CreateSession(u UserID) (*Session, error) {
	tokenStr, err := randomToken(a.tokenLength)
	if err != nil {
		return nil, err
	}
	token := &Session{
		Token:         NewToken(tokenStr),
		User:          u,
		authenticated: false,
		auth:          a,
	}
	a.sessions[SessionID(tokenStr)] = token
	return token, nil
}

func (s *Session) IsAuthenticated() bool {
	return s.authenticated && !s.HasExpired(s.auth.sessionTokenExpiryLimit)
}

func (s *Session) RequestLogin() (LoginToken, error) {
	tokenStr, err := randomToken(s.auth.tokenLength)
	if err != nil {
		return LoginToken{}, err
	}
	token := LoginToken{
		Token: NewToken(tokenStr),
	}
	s.login = token
	s.authenticated = false
	return token, nil
}

func (s *Session) HasValidLoginRequest() bool {
	return s.login.Valid && !s.login.HasExpired(s.auth.loginTokenExpiryLimit)
}

func (s *Session) InvalidateLogin(login LoginID) bool {
	if !s.login.Valid {
		return false
	}
	if s.login.HasExpired(s.auth.loginTokenExpiryLimit) {
		return false
	}
	if string(login) != s.login.Value {
		return false
	}
	s.login.Valid = false
	s.authenticated = true
	return true
}

func (s *Session) RequestCsrf() (CsrfToken, error) {
	tokenStr, err := randomToken(s.auth.tokenLength)
	if err != nil {
		return CsrfToken{}, err
	}
	token := CsrfToken{
		Token: NewToken(tokenStr),
	}
	s.csrf = token
	return token, nil
}

func (s *Session) InvalidateCsrf(csrf CsrfID) bool {
	if !s.csrf.Valid {
		return false
	}
	if s.csrf.HasExpired(s.auth.csrfTokenExpiryLimit) {
		return false
	}
	if string(csrf) != s.csrf.Value {
		return false
	}
	s.csrf.Valid = false
	return true
}

func randomToken(length int) (string, error) {
	bs := make([]byte, length)
	_, err := rand.Read(bs)
	return base64.URLEncoding.EncodeToString(bs), err
}

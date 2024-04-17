package organizer

import (
	"errors"
	"time"
	"crypto/rand"
	"encoding/base64"
	"net/http"
)

type (
	Token struct {
		Value string
		Created time.Time
	}
	LoginToken struct {
		Token
	}
	LoginID string
	SessionToken struct {
		Token
		User UserID
		Authenticated bool
	}
	SessionID string
)

func NewToken(value string) Token {
	return Token{
		Value: value,
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
		loginTokens map[UserID]LoginToken
		sessionIDs map[SessionID]*SessionToken
		tokenLength int
		loginTokenExpiryLimit time.Duration
		sessionTokenExpiryLimit time.Duration
	}
	AuthOpt func(*Authenticator)
)

func NewAuthenticator(opts ...AuthOpt) *Authenticator {
	auth := &Authenticator{
		loginTokens: map[UserID]LoginToken{},
		sessionIDs: map[SessionID]*SessionToken{},
		tokenLength: 50,
		loginTokenExpiryLimit: 10*time.Minute,
		sessionTokenExpiryLimit: time.Hour*24*7,
	}
	for _, opt := range opts {
		opt(auth)
	}
	return auth
}

func WithTokenExpiryLimit(d time.Duration) AuthOpt {
	return func(a *Authenticator) {
		a.loginTokenExpiryLimit = d
	}
}

func WithTokenLength(length int) AuthOpt {
	return func(a *Authenticator) {
		a.tokenLength = length
	}
}

func (a *Authenticator) SessionFromRequest(r *http.Request) (*SessionToken, bool) {
	sessionCookie, err := r.Cookie("session")
	if err != nil {
		return nil, false
	}
	sessionID := SessionID(sessionCookie.Value)
	session, err := a.SessionByID(sessionID)
	if err != nil {
		return nil, false
	}
	return session, true
}

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

func (a *Authenticator) CreateSession(u UserID) (*SessionToken, error) {
	tokenStr, err := randomToken(a.tokenLength)
	if err != nil {
		return nil, err
	}
	token := &SessionToken{
		Token: NewToken(tokenStr),
		User: u,
		Authenticated: false,
	}
	a.sessionIDs[SessionID(tokenStr)] = token
	return token, nil
}

func (a *Authenticator) SessionByID(session SessionID) (*SessionToken, error) {
	t, ok := a.sessionIDs[session]
	if !ok {
		// @todo: app/business logic errors
		return nil, errors.New("session does not exist")
	}
	if t.HasExpired(a.sessionTokenExpiryLimit) {
		delete(a.sessionIDs, session)
		// @todo: app/business logic errors
		return nil, errors.New("session does not exist")
	}
	return t, nil
}

func randomToken(length int) (string, error) {
	bs := make([]byte, length)
	_, err := rand.Read(bs)
	return base64.URLEncoding.EncodeToString(bs), err
}

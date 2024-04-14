package organizer

import (
	"errors"
	"time"
	"crypto/rand"
	"encoding/base64"
)

type (
	LoginToken struct {
		Token string
		Created time.Time
	}
)

type (
	Authenticator struct {
		loginTokens map[UserID]LoginToken
		loginTokenExpiryLimit time.Duration
		loginTokenLength int
	}
	AuthOpt func(*Authenticator)
)

func NewAuthenticator(opts ...AuthOpt) *Authenticator {
	auth := &Authenticator{
		loginTokens: map[UserID]LoginToken{},
		loginTokenExpiryLimit: 10*time.Minute,
		loginTokenLength: 50,
	}
	for _, opt := range opts {
		opt(auth)
	}
	return auth
}

func WithLoginTokenExpiryLimit(d time.Duration) AuthOpt {
	return func(a *Authenticator) {
		a.loginTokenExpiryLimit = d
	}
}

func WithLoginTokenLength(length int) AuthOpt {
	return func(a *Authenticator) {
		a.loginTokenLength = length
	}
}

func (a *Authenticator) CreateLogin(u UserID) (LoginToken, error) {
	tokenStr, err := randomToken(a.loginTokenLength)
	if err != nil {
		var zero LoginToken
		return zero, err
	}
	token := LoginToken{
		Token: tokenStr,
		Created: time.Now(),
	}
	a.loginTokens[u] = token
	return token, nil
}

func (a *Authenticator) ValidateLogin(u UserID) error {
	t, ok := a.loginTokens[u]
	if !ok {
		// @todo: app/business logic errors
		return errors.New("no login token")
	}

	now := time.Now()
	if t.Created.Add(a.loginTokenExpiryLimit).Compare(now) < 0 {
		// @todo: app/business logic errors
		return errors.New("login token expired")
	}

	delete(a.loginTokens, u)
	return nil
}

func randomToken(length int) (string, error) {
	bs := make([]byte, length)
	_, err := rand.Read(bs)
	return base64.URLEncoding.EncodeToString(bs), err
}

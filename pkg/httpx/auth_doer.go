package httpx

import (
	"fmt"
	"net/http"
)

const (
	authKey = "Authorization"
	prefix = "Bearer "
)

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

type authDoer struct {
	parent Doer
	tokenProvider TokenProvider
}

func MakeAuthDoer(parent Doer, provider TokenProvider) Doer {
	return &authDoer{
		parent:        parent,
		tokenProvider: provider,
	}
}

func (d *authDoer) Do(req *http.Request) (*http.Response, error) {
	token, err := d.tokenProvider.GetToken()
	if err != nil {
		return nil, fmt.Errorf("token provider failed, %w", err)
	}

	req.Header.Set(authKey, prefix + token)
	return d.parent.Do(req)
}
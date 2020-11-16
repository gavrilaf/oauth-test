package httpx

import (
	"fmt"
	"net/http"
)

const (
	authKey = "Authorization"
	prefix  = "Bearer "

	retryCount = 1
)

//go:generate mockery --name Doer --outpkg httpxmock --output ./httpxmock --dir .
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

type authDoer struct {
	parent        Doer
	tokenProvider TokenProvider
	metrics       Metrics
}

func MakeAuthDoer(parent Doer, provider TokenProvider, metrics Metrics) Doer {
	return &authDoer{
		parent:        parent,
		tokenProvider: provider,
		metrics:       metrics,
	}
}

func (d *authDoer) Do(req *http.Request) (*http.Response, error) {
	token, err := d.tokenProvider.GetToken()
	if err != nil {
		return nil, fmt.Errorf("token provider failed, %w", err)
	}

	req.Header.Set(authKey, prefix+token)

	var doWithRetry func(attempt int) (*http.Response, error)
	doWithRetry = func(attempt int) (*http.Response, error) {
		resp, err := d.parent.Do(req)
		if err != nil {
			return resp, err
		}

		success, shouldRetry := checkStatusCode(resp.StatusCode)
		if success {
			return resp, nil
		}

		if shouldRetry && attempt < retryCount {
			d.metrics.RecordCount("doer-retry-count")
			return doWithRetry(attempt + 1)
		}

		return resp, err
	}

	return doWithRetry(0)
}

// Check the response status code
func checkStatusCode(code int) (bool, bool) {
	return code >= 200 && code < 300, code == 401
}

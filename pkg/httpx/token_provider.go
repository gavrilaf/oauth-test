package httpx

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
)

//go:generate mockery --name TokenProvider --outpkg httpxmock --output ./httpxmock --dir .
type TokenProvider interface {
	GetToken() (string, error)
	IsTokenValid() bool
	ForceRefresh() error
}

func MakeTokenProvider(authUrl string, metrics Metrics) TokenProvider {
	return &tokenProvider{
		authUrl: authUrl,
		client:  http.DefaultClient,
		lock:    &sync.Mutex{},
		metrics: metrics,
	}
}

// impl

type tokenProvider struct {
	authUrl  string
	client   *http.Client
	lock     *sync.Mutex
	token    string
	expireAt time.Time
	lifetime time.Duration
	metrics  Metrics
}

func (s *tokenProvider) GetToken() (string, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	valid := s.isTokenValid()

	if valid {
		return s.token, nil
	}

	fmt.Println("token invalid or expired")
	if err := s.refreshToken(); err != nil {
		return "", err
	}

	return s.token, nil
}

func (s *tokenProvider) IsTokenValid() bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.isTokenValid()
}

func (s *tokenProvider) ForceRefresh() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.refreshToken()
}

var retryLogic = backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 3)

func (s *tokenProvider) refreshToken() error {
	update := func() error {
		token, err := s.readToken()
		if err != nil {
			fmt.Printf("read token error: %v\n", err)
			s.metrics.RecordCount("token-read-failed")
			return err
		}

		s.token = token.Token
		s.lifetime = time.Duration(token.Expire)
		s.expireAt = TimeNow().Add(time.Duration(token.Expire) * time.Second)

		return nil
	}

	err := backoff.Retry(update, retryLogic)
	if err != nil {
		s.token = ""
		s.lifetime = 0
		s.expireAt = time.Time{}
		s.metrics.RecordCount("token-refresh-failed")

		return fmt.Errorf("failed to refresh token, %w", err)
	}

	s.metrics.RecordCount("token-refreshed")
	fmt.Println("token refreshed")
	return nil
}

func (s *tokenProvider) isTokenValid() bool {
	return s.token != "" && !s.expired()
}

func (s *tokenProvider) expired() bool {
	if s.expireAt.IsZero() {
		return false
	}

	return s.expireAt.Add(-s.lifetime).Before(TimeNow())
}

func (s *tokenProvider) readToken() (Token, error) {
	resp, err := s.client.Get(s.authUrl)
	if err != nil {
		return Token{}, fmt.Errorf("token request error, %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Token{}, fmt.Errorf("failed to read token body, %w", err)
	}

	if resp.StatusCode != 200 {
		return Token{}, fmt.Errorf("token request failed, %d, %s", resp.StatusCode, string(body))
	}

	var token Token
	if err := json.Unmarshal(body, &token); err != nil {
		return Token{}, fmt.Errorf("invalid token, %w", err)
	}

	return token, nil
}

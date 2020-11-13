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

type TokenProvider interface {
	GetToken() (string, error)
	IsTokenValid() bool
	ForceRefresh() error
}

func MakeTokenProvider(authUrl string) TokenProvider {
	return &tokenProvider{
		authUrl: authUrl,
		client:  http.DefaultClient,
		lock:    &sync.RWMutex{},
	}
}

type tokenProvider struct {
	authUrl  string
	client   *http.Client
	lock     *sync.RWMutex
	token    string
	expireAt time.Time
	lifetime time.Duration
}

func (s *tokenProvider) GetToken() (string, error) {
	token, valid := func() (string, bool) {
		s.lock.RLock()
		defer s.lock.RUnlock()
		return s.token, s.isTokenValid()
	}()

	if valid {
		return token, nil
	}

	fmt.Println("token invalid or expired")
	if err := s.refreshToken(); err != nil {
		return "", err
	}

	return s.token, nil
}

func (s *tokenProvider) IsTokenValid() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.isTokenValid()
}

func (s *tokenProvider) ForceRefresh() error {
	return s.refreshToken()
}

var retryLogic = backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 3)

func (s *tokenProvider) refreshToken() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	fmt.Println("refreshing token")
	if s.isTokenValid() { // updated in another goroutine
		fmt.Println("already refreshed")
		return nil
	}

	update := func() error {
		token, err := s.readToken()
		if err != nil {
			fmt.Printf("read token error: %v\n", err)
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

		return fmt.Errorf("failed to refresh token, %w", err)
	}

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

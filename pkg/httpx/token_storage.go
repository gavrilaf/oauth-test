package httpx

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

type TokenStorage interface {
	GetToken() (string, error)
}

func MakeTokenStorage(authUrl string) TokenStorage {
	return &tokenStorage{
		authUrl: authUrl,
		client:  http.DefaultClient,
		lock:    &sync.RWMutex{},
	}
}

type tokenStorage struct {
	authUrl  string
	client   *http.Client
	lock     *sync.RWMutex
	token    string
	expireAt time.Time
	lifetime time.Duration
}

func (s *tokenStorage) GetToken() (string, error) {

	token, valid := func() (string, bool) {
		s.lock.RLock()
		defer s.lock.RUnlock()
		return s.token, s.isTokenValid()
	}()

	if valid {
		fmt.Println("token is valid, use it")
		return token, nil
	}

	fmt.Println("refresh token")
	if err := s.refreshToken(); err != nil {
		fmt.Printf("failed to refresh token, %v", err)
		return "", err
	}

	fmt.Println("token refreshed")

	return s.token, nil
}

func (s *tokenStorage) refreshToken() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.isTokenValid() { // update in another goroutine
		return nil
	}

	token, err := s.readToken()
	if err != nil {
		return fmt.Errorf("failed to refresh token, %w", err)
	}

	s.token = token.Token
	s.lifetime = time.Duration(token.Expire)
	s.expireAt = time.Now().Add(time.Duration(token.Expire) * time.Second)

	return nil
}

func (s *tokenStorage) isTokenValid() bool {
	return s.token != "" && !s.expired()
}

func (s *tokenStorage) expired() bool {
	if s.expireAt.IsZero() {
		return false
	}

	return s.expireAt.Add(-s.lifetime).Before(TimeNow())
}

func (s *tokenStorage) readToken() (Token, error) {
	resp, err := s.client.Get(s.authUrl)
	if err != nil {
		return Token{}, fmt.Errorf("token request error, %w", err)
	}

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

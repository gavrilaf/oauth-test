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

type TokenEvent int

const (
	TokenEventNeedRefresh TokenEvent = iota
	TokenEventRefreshed
	TokenEventForceRefresh
	TokenEventReadError
	TokenEventRefreshFailed
)

type TokenMetricsDelegate func(event TokenEvent, err error)

//go:generate mockery --name TokenProvider --outpkg httpxmock --output ./httpxmock --dir .
type TokenProvider interface {
	GetToken() (string, error)
	IsTokenValid() bool
	ForceRefresh() error

	StartAutoRefresh()
	StopAutoRefresh()
}

func MakeTokenProvider(authUrl string, metrics TokenMetricsDelegate) TokenProvider {
	return &tokenProvider{
		authUrl: authUrl,
		client:  http.DefaultClient,
		closed:  make(chan struct{}),
		metrics: metrics,
	}
}

// impl

type tokenProvider struct {
	authUrl  string
	client   *http.Client
	lock     sync.Mutex
	token    string
	expireAt time.Time
	lifetime time.Duration
	closed   chan struct{}
	metrics  TokenMetricsDelegate
}

func (p *tokenProvider) GetToken() (string, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	valid := p.isTokenValid()

	if valid {
		return p.token, nil
	}

	if p.metrics != nil {
		p.metrics(TokenEventNeedRefresh, nil)
	}

	if err := p.refreshToken(); err != nil {
		return "", err
	}

	return p.token, nil
}

func (p *tokenProvider) IsTokenValid() bool {
	p.lock.Lock()
	defer p.lock.Unlock()

	return p.isTokenValid()
}

func (p *tokenProvider) ForceRefresh() error {
	p.lock.Lock()
	defer p.lock.Unlock()

	err := p.refreshToken()
	if p.metrics != nil {
		p.metrics(TokenEventForceRefresh, err)
	}

	return err
}

const (
	refreshBiasTime   = 2 * time.Second
	defaultWakeupTime = 10 * time.Second
)

func (p *tokenProvider) StartAutoRefresh() {
	// returns: ('should refresh token', 'next wake up time')
	checkToken := func() (bool, time.Duration) {
		if !p.isTokenValid() {
			return true, defaultWakeupTime
		}

		estimatedExpiration := p.expireAt.Add(p.lifetime - refreshBiasTime)
		notExpired := estimatedExpiration.Before(TimeNow())
		if notExpired {
			return false, TimeNow().Sub(estimatedExpiration)
		} else {
			return true, p.lifetime - refreshBiasTime
		}
	}

	go func() {
		for {
			p.lock.Lock()

			shouldRefresh, nextWakeUp := checkToken()
			if shouldRefresh {
				_ = p.refreshToken()
			}

			p.lock.Unlock()

			select {
			case <-time.After(nextWakeUp):
				break
			case <-p.closed:
				return
			}
		}
	}()
}

func (p *tokenProvider) StopAutoRefresh() {
	close(p.closed)
}

// private

var retryLogic = backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 3)

func (p *tokenProvider) refreshToken() error {
	update := func() error {
		token, err := p.readToken()
		if err != nil {
			if p.metrics != nil {
				p.metrics(TokenEventReadError, err)
			}
			return err
		}

		p.token = token.Token
		p.lifetime = time.Duration(token.Expire)
		p.expireAt = TimeNow().Add(time.Duration(token.Expire) * time.Second)

		return nil
	}

	err := backoff.Retry(update, retryLogic)
	if err != nil {
		p.token = ""
		p.lifetime = 0
		p.expireAt = time.Time{}
		if p.metrics != nil {
			p.metrics(TokenEventRefreshFailed, err)
		}

		return fmt.Errorf("failed to refresh token, %w", err)
	}

	if p.metrics != nil {
		p.metrics(TokenEventRefreshed, nil)
	}

	return nil
}

func (p *tokenProvider) isTokenValid() bool {
	return p.token != "" && !p.expired()
}

func (p *tokenProvider) expired() bool {
	if p.expireAt.IsZero() {
		return false
	}

	return p.expireAt.Add(-p.lifetime).Before(TimeNow())
}

func (p *tokenProvider) readToken() (Token, error) {
	resp, err := p.client.Get(p.authUrl)
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

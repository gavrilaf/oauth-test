package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testTime(sec int) time.Time {
	return time.Date(2020, time.April, 1, 10, 10, sec, 0, time.UTC)
}

func TestTokenProvider_GetToken(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{\"token\": \"aaaa\", \"expire\": 300}"))
		}

		server := httptest.NewServer(http.HandlerFunc(handler))
		defer server.Close()

		provider := MakeTokenProvider(server.URL, nil)

		token, err := provider.GetToken()
		assert.NoError(t, err)
		assert.Equal(t, "aaaa", token)

		assert.True(t, provider.IsTokenValid())
	})

	t.Run("get token succeeded after 2 retries", func(t *testing.T) {
		retryCounter := 0
		handler := func(w http.ResponseWriter, r *http.Request) {
			retryCounter += 1
			if retryCounter == 2 {
				w.Write([]byte("{\"token\": \"aaaa\", \"expire\": 300}"))
			} else {
				w.WriteHeader(500)
			}
		}

		server := httptest.NewServer(http.HandlerFunc(handler))
		defer server.Close()

		provider := MakeTokenProvider(server.URL, nil)

		token, err := provider.GetToken()
		assert.NoError(t, err)
		assert.Equal(t, "aaaa", token)

		assert.True(t, provider.IsTokenValid())

		assert.Equal(t, 2, retryCounter)
	})

	t.Run("get token failed", func(t *testing.T) {
		retryCounter := 0
		handler := func(w http.ResponseWriter, r *http.Request) {
			retryCounter += 1
			w.WriteHeader(500)
		}

		server := httptest.NewServer(http.HandlerFunc(handler))
		defer server.Close()

		provider := MakeTokenProvider(server.URL, nil)

		token, err := provider.GetToken()

		assert.Error(t, err)
		assert.Empty(t, token)

		assert.False(t, provider.IsTokenValid())

		assert.Equal(t, 4, retryCounter)
	})

	t.Run("check token expiration", func(t *testing.T) {
		secondsNow := 1

		timeNow = func() time.Time {
			return testTime(secondsNow)
		}

		callsCounter := 0
		handler := func(w http.ResponseWriter, r *http.Request) {
			if callsCounter == 0 {
				w.Write([]byte("{\"token\": \"aaaa\", \"expire\": 10}"))
			} else {
				w.Write([]byte("{\"token\": \"bbbb\", \"expire\": 10}"))
			}
			callsCounter += 1
		}

		server := httptest.NewServer(http.HandlerFunc(handler))
		defer server.Close()

		provider := MakeTokenProvider(server.URL, nil)

		token, err := provider.GetToken()
		assert.NoError(t, err)
		assert.Equal(t, "aaaa", token)

		t.Run("get token after 7 seconds, should be the same", func(t *testing.T) {
			secondsNow += 7
			token, err := provider.GetToken()
			assert.NoError(t, err)
			assert.Equal(t, "aaaa", token)
		})

		t.Run("get token after 11 seconds, should be updated", func(t *testing.T) {
			secondsNow += 11
			token, err := provider.GetToken()
			assert.NoError(t, err)
			assert.Equal(t, "bbbb", token)
		})

		assert.Equal(t, 2, callsCounter)

		timeNow = time.Now
	})
}

func TestTokenProvider_IsTokenValid(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("{\"token\": \"aaaa\", \"expire\": 10}"))
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	secondsNow := 1
	timeNow = func() time.Time {
		return testTime(secondsNow)
	}

	provider := MakeTokenProvider(server.URL, nil)

	t.Run("token is invalid from the beginning", func(t *testing.T) {
		assert.False(t, provider.IsTokenValid())
	})

	t.Run("token is valid when refreshed", func(t *testing.T) {
		_, err := provider.GetToken()
		assert.NoError(t, err)
		assert.True(t, provider.IsTokenValid())
	})

	t.Run("token is invalid when expired", func(t *testing.T) {
		secondsNow += 11
		assert.False(t, provider.IsTokenValid())
	})

	timeNow = time.Now
}

func TestTokenProvider_ForceRefresh(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("{\"token\": \"aaaa\", \"expire\": 10}"))
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	provider := MakeTokenProvider(server.URL, nil)

	t.Run("token is valid after force refseh", func(t *testing.T) {
		assert.False(t, provider.IsTokenValid())

		err := provider.ForceRefresh()
		assert.NoError(t, err)

		assert.True(t, provider.IsTokenValid())
	})
}

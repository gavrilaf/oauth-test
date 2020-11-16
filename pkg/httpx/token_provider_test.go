package httpx_test

import (
	"github.com/gavrilaf/oauth-test/pkg/httpx/httpxmock"
	"github.com/stretchr/testify/mock"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/gavrilaf/oauth-test/pkg/httpx"
)

func testTime(sec int) time.Time {
	return time.Date(2020, time.April, 1, 10, 10, sec, 0, time.UTC)
}

func TestTokenProvider_GetToken(t *testing.T) {
	metrics := &httpxmock.Metrics{}
	metrics.On("RecordCount", mock.Anything)

	t.Run("happy path", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{\"token\": \"aaaa\", \"expire\": 300}"))
		}

		server := httptest.NewServer(http.HandlerFunc(handler))
		defer server.Close()

		provider := httpx.MakeTokenProvider(server.URL, metrics)

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

		provider := httpx.MakeTokenProvider(server.URL, metrics)

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

		provider := httpx.MakeTokenProvider(server.URL, metrics)

		token, err := provider.GetToken()

		assert.Error(t, err)
		assert.Empty(t, token)

		assert.False(t, provider.IsTokenValid())

		assert.Equal(t, 4, retryCounter)
	})

	t.Run("check token expiration", func(t *testing.T) {
		secondsNow := 1

		httpx.TimeNow = func() time.Time {
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

		provider := httpx.MakeTokenProvider(server.URL, metrics)

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

		httpx.TimeNow = time.Now
	})
}

package main

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"

	"github.com/gavrilaf/oauth-test/pkg/httpx"
	"github.com/gavrilaf/oauth-test/pkg/log"
)

const (
	tokenLifetime = 10

	authErrorRate = 0.3
	doErrorRate   = 0.3
)

func bernoulliTryFail(rate float64) bool {
	return rand.Float64() < rate
}

func main() {
	log.InitLog()

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.Debug = true

	rand.Seed(time.Now().UnixNano())

	signingKey := []byte("secret-key")

	e.GET("/auth", func(c echo.Context) error {
		if bernoulliTryFail(authErrorRate) {
			return c.NoContent(500)
		}

		claims := &jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Second * tokenLifetime).Unix(),
			Issuer:    "provider",
		}

		t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		ss, err := t.SignedString(signingKey)
		if err != nil {
			return echo.NewHTTPError(http.StatusConflict, err.Error())
		}

		token := httpx.Token{
			Expire: tokenLifetime,
			Token:  ss,
		}

		return c.JSON(200, token)
	})

	jwtConfig := middleware.JWTConfig{
		SigningKey: signingKey,
	}

	handler := func(c echo.Context) error {
		if bernoulliTryFail(doErrorRate) {
			return c.NoContent(401)
		}

		return c.JSON(200, map[string]string{"result": "success"})
	}

	e.GET("/do", handler, middleware.JWTWithConfig(jwtConfig))

	log.L.Info("Starting provider")

	e.Logger.Fatal(e.Start(":7575"))
}

package main

import (
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"

	"github.com/gavrilaf/oauth-test/pkg/log"
)

type Token struct {
	Expire int    `json:"expire"`
	Token  string `json:"token"`
}

func main() {
	log.InitLog()

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.Debug = true

	signingKey := []byte("secret-key")

	e.GET("/auth", func(c echo.Context) error {
		claims := &jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Second * 30).Unix(),
			Issuer:    "provider",
		}

		t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		ss, err := t.SignedString(signingKey)
		if err != nil {
			return echo.NewHTTPError(http.StatusConflict, err.Error())
		}

		token := Token{
			Expire: 300,
			Token:  ss,
		}

		return c.JSON(200, token)
	})

	jwtConfig := middleware.JWTConfig{
		SigningKey:    signingKey,
	}

	handler := func(c echo.Context) error {
		return c.JSON(200, map[string]string{"result": "sucess"})
	}

	e.GET("/do", handler, middleware.JWTWithConfig(jwtConfig))

	log.L.Info("Starting provider")

	e.Logger.Fatal(e.Start(":7575"))
}

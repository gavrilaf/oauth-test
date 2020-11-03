package main

import (
	"github.com/labstack/echo"

	"github.com/gavrilaf/oauth-test/pkg/log"
)

func main() {
	log.InitLog()

	e := echo.New()

	e.GET("/auth", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"a": "a"})
	})


	log.L.Info("Starting provider")

	e.Logger.Fatal(e.Start(":7575"))
}

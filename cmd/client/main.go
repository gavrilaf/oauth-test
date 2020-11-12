package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gavrilaf/oauth-test/pkg/httpx"
)

func run(id int, provider httpx.TokenProvider) {
	doer := httpx.MakeAuthDoer(http.DefaultClient, provider)

	for {
		req, err := http.NewRequest("GET", "http://localhost:7575/do", nil)
		if err != nil {
			panic(err.Error())
		}

		resp, err := doer.Do(req)
		if err != nil {
			fmt.Printf("%d failed\n", id)
		} else {
			fmt.Printf("%d success\n", id)
		}

		resp.Body.Close()

		time.Sleep(time.Second)
	}
}

func main() {
	provider := httpx.MakeTokenProvider("http://127.0.0.1:7575/auth")

	go run(1, provider)
	go run(2, provider)
	go run(3, provider)

	for {}
}

package main

import (
	"fmt"
	"github.com/gavrilaf/oauth-test/pkg/httpx"
)

func main() {

	stg := httpx.MakeTokenStorage("http://127.0.0.1:7575/auth")
	token, err := stg.GetToken()
	if err != nil {
		fmt.Printf("failed to read token: %v\n", err)
	} else {
		fmt.Printf("token: %s\n", token)
	}
}

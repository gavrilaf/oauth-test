package main

import (
	"encoding/json"
	"fmt"
	"github.com/gavrilaf/oauth-test/pkg/httpx"
	"io/ioutil"
	"net/http"

	"github.com/gavrilaf/oauth-test/pkg/log"

)

func main() {
	client := http.DefaultClient

	resp, err := client.Get("http://127.0.0.1:7575/auth")
	if err != nil {
		log.L.WithError(err).Fatal("failed to auth")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.L.WithError(err).Fatal("failed to read auth body")
	}

	var token httpx.Token
	if err := json.Unmarshal(body, &token); err != nil {
		log.L.WithError(err).Fatal("failed to unmarshal token")
	}

	fmt.Println(token)
}

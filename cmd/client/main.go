package main

import (
	"fmt"
	"github.com/gavrilaf/oauth-test/pkg/log"
	"io/ioutil"
	"net/http"
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

	fmt.Println(string(body))
}

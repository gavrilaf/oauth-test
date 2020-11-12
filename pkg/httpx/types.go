package httpx

import "time"

var TimeNow = time.Now

type Token struct {
	Expire int    `json:"expire"`
	Token  string `json:"token"`
}


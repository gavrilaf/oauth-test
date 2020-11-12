package httpx

type Token struct {
	Expire int    `json:"expire"`
	Token  string `json:"token"`
}


SHELL := /bin/zsh

pb:
	go build -a -o ./bin/provider ./cmd/provider
	chmod +x ./bin/provider

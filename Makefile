SHELL := /bin/zsh

pb:
	go build -a -o ./bin/provider ./cmd/provider
	chmod +x ./bin/provider

pr:
	go build -a -o ./bin/provider ./cmd/provider
	chmod +x ./bin/provider
	./bin/provider

cb:
	go build -a -o ./bin/client ./cmd/client
	chmod +x ./bin/client

cr:
	go build -a -o ./bin/client ./cmd/client
	chmod +x ./bin/client
	./bin/client

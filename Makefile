build:
	go build

run: build
	sudo ./go-container alpine /bin/sh

.PHONY: build run

build:
	go build

run: build
	sudo ./go-container alpine

.PHONY: build run

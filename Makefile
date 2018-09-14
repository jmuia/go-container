build:
	go build

run: build
	sudo ./go-container

.PHONY: build run

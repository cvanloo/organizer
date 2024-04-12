.PHONY: build run debug

.DEFAULT: build

build: .FORCE
	go build ./...

server: .FORCE
	go build cmd/server.go

run: server
	sudo -u organizer ./server

debug: .FORCE
	dlv debug cmd/server.go

.FORCE:

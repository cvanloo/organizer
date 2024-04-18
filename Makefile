.PHONY: build run debug

.DEFAULT: build

build: .FORCE
	go build ./...

server: .FORCE
	go build -tags=delve cmd/server.go

run: server
	sudo -u organizer ./server

debug: .FORCE
	dlv debug --build-flags="-tags=delve" cmd/server.go

release: .FORCE
	go build cmd/server.go

.FORCE:

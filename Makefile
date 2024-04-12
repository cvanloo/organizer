.PHONY: build run debug

.DEFAULT: build

build: .FORCE
	go build github.com/cvanloo/organizer

run: .FORCE
	go run github.com/cvanloo/organizer/cmd/server

debug: .FORCE
	dlv debug github.com/cvanloo/organizer/cmd/server

.FORCE:

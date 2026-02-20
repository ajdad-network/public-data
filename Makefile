.PHONY: test run all

all: test run

test:
	go test -v ./cmd/

run: test
	go run ./cmd/amalgamate

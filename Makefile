IMAGE := pgaxatte/rflink-prom:latest
BIN := rflink-prom

docker:
	docker build -t $(IMAGE) .

build:
	go build -o $(BIN) ./...

run:
	go run ./...

.PHONY: docker build run

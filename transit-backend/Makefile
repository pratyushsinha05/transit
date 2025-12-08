build:
	go build -o server ./cmd/server

run:
	go run ./cmd/server

test:
	go test ./...

clean:
	rm -f server

docker-build:
	docker build -t transit-backend:latest .

run:
	nodemon --exec go run ./cmd/main.go --signal SIGTERM

document:
	swag init -g ./cmd/main.go

test-all:
	go test -v ./...

test-user:
	go test ./controllers/userhandler_test.go

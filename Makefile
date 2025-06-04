servers:
	nodemon --exec go run ./cmd/main.go --signal SIGTERM

document:
	swag init -g ./cmd/main.go
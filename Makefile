# Run golangci-lint to lint the whole codebase, ensuring code quality
lint:
	go mod tidy
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.58.1 run

build:
	go build -o ./build/node-automated-deployer .

run:
	go run main.go compose > config/docker-compose.yaml

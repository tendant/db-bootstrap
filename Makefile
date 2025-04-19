# Variables
BINARY_NAME=dbstrap
CMD_PATH=cmd/dbstrap
BUILD_DIR=bin
VERSION ?= $(shell git describe --tags --always --dirty)

# Default target
all: build

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_PATH)

clean:
	rm -rf $(BUILD_DIR)

run:
	./$(BUILD_DIR)/$(BINARY_NAME) run --config-path=samples/bootstrap.yaml

build-static:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_PATH)

# Updated buildx-based Docker image build
docker-build:
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--push \
		--tag wang/dbstrap:$(VERSION) \
		--tag wang/dbstrap:latest \
		.

docker-run:
	docker run --rm \
		-e DATABASE_URL=$(DATABASE_URL) \
		-e TEST_USER_PASSWORD=$(TEST_USER_PASSWORD) \
		-v $(PWD)/samples/bootstrap.yaml:/bootstrap.yaml \
		$(BINARY_NAME):$(VERSION) run --config-path=/bootstrap.yaml

install:
	go install github.com/tendant/dbstrap/$(CMD_PATH)@latest

test:
	go test ./...

.PHONY: all build clean run build-static docker-build docker-run install test

GIT_COMMIT = $(shell git rev-parse HEAD)
GO_SOURCE_FILES = $(shell find pkg -type f -name "*.go")

build: vendor $(GO_SOURCE_FILES)
	go build -i -ldflags "-X main.GitCommit=${GIT_COMMIT} -extldflags '-static'" -o resource-helper ./pkg

mont: mont.go
	go build -i -ldflags "-X main.GitCommit=${GIT_COMMIT} -extldflags '-static'" -o mont mont.go

vendor:
	glide up -v

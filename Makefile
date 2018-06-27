GIT_COMMIT = $(shell git rev-parse HEAD)
GO_SOURCE_FILES = $(shell find pkg -type f -name "*.go")

build: $(GO_SOURCE_FILES)
	go build -i -ldflags \
		"-X main.GitCommit=${GIT_COMMIT} -extldflags '-static'" \
		-o kube-resource-explorer ./cmd/kube-resource-explorer

vendor:
	glide up -v

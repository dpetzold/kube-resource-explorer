GIT_COMMIT = $(shell git rev-parse HEAD)
GO_SOURCE_FILES = $(shell find pkg -type f -name "*.go")


build: $(GO_SOURCE_FILES)
	go build -i -ldflags \
		"-X main.GitCommit=${GIT_COMMIT} -extldflags '-static'" \
		-o kube-resource-explorer ./cmd/kube-resource-explorer

vendor:
	glide up -v


docker-build:
	docker build --rm -t dpetzold/kube-resource-explorer .
	docker push dpetzold/kube-resource-explorer


run:
	docker run --rm -it \
		-v${HOME}/.kube:/.kube \
		-v${HOME}/.config/gcloud:/.config/gcloud \
		-v/etc/ssl/certs:/etc/ssl/certs \
		dpetzold/kube-resource-explorer

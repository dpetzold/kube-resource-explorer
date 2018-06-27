FROM golang:1.10-alpine AS builder
RUN apk add --no-cache git make curl \
  && (curl https://glide.sh/get | sh)

ARG BASEDIR=/go/src/github.com/dpetzold/kube-resource-explorer/

RUN mkdir -p ${BASEDIR}
WORKDIR ${BASEDIR}

ENV CGO_ENABLED=0
ENV GOOS=linux

COPY glide.* ${BASEDIR}
RUN glide install -v

COPY . .
RUN  make build

FROM scratch
COPY --from=builder /tmp /tmp
COPY --from=builder /go/src/github.com/dpetzold/kube-resource-explorer/kube-resource-explorer /

ENTRYPOINT ["/kube-resource-explorer"]

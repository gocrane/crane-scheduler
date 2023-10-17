ARG PKGNAME

# Build the manager binary
FROM golang:1.19.9-alpine as builder

ARG LDFLAGS
ARG PKGNAME

WORKDIR /go/src/github.com/gocrane/crane-scheduler

# Add build deps
RUN apk add build-base

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN unset https_proxy HTTPS_PROXY HTTP_PROXY http_proxy && go mod download

# Copy the go source
COPY pkg pkg/
COPY cmd cmd/

# Build
RUN go build -ldflags="${LDFLAGS}" -a -o ${PKGNAME} /go/src/github.com/gocrane/crane-scheduler/cmd/${PKGNAME}/main.go

FROM alpine:latest
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories
RUN unset https_proxy HTTPS_PROXY HTTP_PROXY http_proxy && apk add -U tzdata

WORKDIR /
ARG PKGNAME
COPY --from=builder /go/src/github.com/gocrane/crane-scheduler/${PKGNAME} .

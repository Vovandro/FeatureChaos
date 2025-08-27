# syntax=docker/dockerfile:1

## Build
FROM golang:1.23.4-alpine AS builder

WORKDIR /go/src/app

RUN apk add --no-cache make build-base

ENV GOPATH /go

COPY go.mod .
COPY go.sum .
RUN go mod download && go mod verify

COPY . .

ENV GOPATH /go
ENV PATH $PATH:/go/bin:$GOPATH/bin

# Build service binary
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags '-w -s' -tags musl -o ./FeatureChaos ./cmd/app/main.go

# Install goose CLI (for running DB migrations)
RUN GOBIN=/go/bin go install github.com/pressly/goose/v3/cmd/goose@v3.19.1

## Deploy
FROM alpine:3.19 as runtime

WORKDIR /app

COPY --from=builder /go/src/app/FeatureChaos ./
COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY migrations ./migrations

# For TLS DB connections if using postgres URLs with sslmode != disable
RUN apk add --no-cache ca-certificates make

WORKDIR /app

ENTRYPOINT [ "./FeatureChaos" ]

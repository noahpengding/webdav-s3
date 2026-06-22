# syntax=docker/dockerfile:1

FROM dhi.io/golang:1-debian-dev AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN test -z "$(gofmt -l .)"
RUN go test ./...
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/webdav .

FROM alpine:latest

WORKDIR /app
RUN apk add --no-cache gcompat
RUN apk add --no-cache ca-certificates \
	&& addgroup -S app \
	&& adduser -S -G app app
COPY --from=builder /out/webdav /app/webdav

USER app
ENTRYPOINT ["/app/webdav"]

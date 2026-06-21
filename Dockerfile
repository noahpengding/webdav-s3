FROM dhi.io/golang:1-debian-dev AS builder

WORKDIR /app
COPY . /app
RUN gofmt -l .
RUN go get -d -v
RUN go build -o webdav -v .

FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache gcompat
COPY --from=builder /app/webdav /app/webdav
CMD [ "/app/webdav" ]

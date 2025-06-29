FROM golang:1.22.1-alpine As builder

WORKDIR /app
COPY . /app
RUN gofmt -l .
RUN go get -d -v
RUN go build -o webdav -v .

FROM alpine:3.14.2
WORKDIR /app
RUN mkdir /app/conf
COPY --from=builder /app/webdav /app/webdav
COPY --from=builder /app/config_sample.yaml /app/config_sample.yaml
CMD [ "/app/webdav" ]
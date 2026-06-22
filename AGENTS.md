# AGENTS.md

## Project Overview

This is a small Go service that exposes a WebDAV-like HTTP interface backed by an S3 bucket. It is a single `main` package with no framework. The production artifact is a Docker image, and GitHub Actions builds the image for pull requests and pushes a multi-arch image from `main`.

## Important Files

- `main.go`: loads configuration, optionally starts `net/http/pprof`, and starts the HTTP server.
- `config.go`: reads lowercase environment variables. There is no active config file loader.
- `s3-client.go`: wraps AWS SDK for Go v2 S3 operations, path-style endpoint setup, content-type inference, and paginated object listing.
- `webdav.go`: HTTP/WebDAV handlers for `GET`, `HEAD`, `PROPFIND`, `PUT`, `DELETE`, `COPY`, `MOVE`, `MKCOL`, and `OPTIONS`.
- `log.go`: tiny log-level helper controlled by `loglevel`.
- `Dockerfile`: module-aware multi-stage build. It runs `gofmt`, `go test ./...`, then builds a static Linux binary into an Alpine runtime.
- `.github/workflows/build.yml`: PRs build the image without pushing; pushes to `main` publish `linux/amd64` and `linux/arm64` images to GHCR.

## Configuration

Required for normal operation:

- `access_key`: S3 access key.
- `secret_key`: S3 secret key.
- `bucket_name`: target bucket.
- `region`: S3 signing region. For custom endpoints, use a signing region such as `us-east-1`; if only `endpoint` is set, the code defaults this to `us-east-1`.

Optional:

- `endpoint`: custom S3-compatible endpoint. If omitted, the AWS regional endpoint is used.
- `loglevel`: `INFO` by default. Set `debug` for request/S3 debug logs.
- `port`: HTTP port, default `8080`.
- `baseurl`: currently logged only.
- `pprof_addr`: opt-in pprof listener, for example `127.0.0.1:6060`.

## Common Commands

```bash
gofmt -w *.go
go test ./...
docker build -t webdav-s3 .
```

Run locally after setting the required environment variables:

```bash
go run .
```

Run the container:

```bash
docker run --rm -p 8080:8080 --env-file .env webdav-s3
```

## Design Notes

- S3 has no real directories. Directory behavior is modeled with prefixes and `Delimiter: "/"`; `MKCOL` writes an empty object with a trailing slash.
- `ListObjectsV2` is paginated in `s3-client.go`; avoid replacing it with a single-page call.
- Handlers pass `r.Context()` into S3 operations. Preserve this so client disconnects and timeouts can cancel upstream calls.
- `PUT` currently reads the full request body into memory so it can infer content type before uploading. This is the highest-value runtime modernization target for large files.
- The server has no built-in authentication. Deploy it behind a reverse proxy or load balancer that provides auth if the bucket contents are private.
- `COPY` and `MOVE` use S3 copy/delete operations and may need extra care for unusual keys because S3 copy source encoding is strict.

## Safe Modernization Targets

- Replace full-buffer `PUT` with multipart streaming using AWS SDK v2 upload manager while preserving content-type behavior.
- Add integration tests against MinIO or LocalStack for WebDAV method behavior.
- Replace the custom logger with `log/slog`.
- Add graceful shutdown and configurable read/write timeouts.
- Add optional basic auth or document the expected reverse-proxy auth contract more formally.

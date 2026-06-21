# WebDAV Server with AWS S3 Backend

A lightweight GO WebDAV server that uses AWS S3 as a backend. It uses AWS SDK for Go to interact with S3, which means no storage is required on the server itself. Configuration is provided through environment variables.


## Installing

### From Source

1. Clone the repository and install the dependencies:
```bash
git clone https://github.com/Noahdingpeng/webdav-s3
cd webdav-s3
go get -d -v
go build -o webdav -v .
```
2. Set the required environment variables:
```bash
export loglevel=INFO
export region=us-east-1
export access_key=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
export secret_key=bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
export bucket_name=webdav
export endpoint=https://s3.us-east-1.amazonaws.com
export baseurl=http://127.0.0.1
export port=8080
```
3. Run the server:
```bash
./webdav
```

### Docker Compose
```yaml
services:
  webdav:
    image: noahding1214/webdav-s3:latest
    container_name: webdav
    restart: always
    environment:
        - loglevel=INFO
        - region=us-east-1
        - access_key=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
        - secret_key=bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
        - bucket_name=webdav
        - endpoint=https://s3.us-east-1.amazonaws.com
        - baseurl=http://127.0.0.1
        - port=8080
    ports:
        - 8080:8080
```

### Docker Compose + Traefik
```yaml
services:
  webdav:
    image: noahding1214/webdav-s3:latest
    container_name: webdav
    restart: always
    env_file:
        - .env
    labels:
        # Change the domain name and ports to your own environment values
        - "traefik.enable=true"
        - "traefik.http.routers.webdav.rule=Host(`webdav.example.com`)"
        - "traefik.http.routers.webdav.entrypoints=websecure"
        - "traefik.http.services.webdav.loadbalancer.server.port=8080"
```

### Nginx Proxy Reverse
```nginx
location / {
    proxy_set_header Host $http_host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_connect_timeout 300;
    proxy_http_version 1.1;
    proxy_set_header Connection "";
    chunked_transfer_encoding off;

    proxy_pass http://127.0.0.1:8080;

    ## Add Basic Auth if needed
    auth_basic "Restricted Access!";
    auth_basic_user_file /etc/nginx/.htpasswd;
}
```

## License
This project is licensed under the MIT License - see the LICENSE.md file for details

## Todo Life
- [x] Basic WebDAV Server with S3 Backend
- [x] GET, PUT, DELETE, MKCOL, COPY, MOVE, OPTIONS, PROPFIND, Head Methods
- [ ] Inside Basic AUTH
- [X] Use Environment Variables for Configuration
- [ ] Upgrade AWS-SDK to AWS Go SDK v2 for large file upload & download
- [ ] GitHub Actions for CI/CD
- [X] Better Logging and Error Handling

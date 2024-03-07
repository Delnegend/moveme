# Moveme

Simple HTTP redirect server, simple configuration.

## Build from source
- Minimum Go version? Just use lastest.
- Clone the repository and cd int.
- `go get github.com/lmittmann/tint`
- Run `go build .`

## Docker compose example
```yaml
version: "3"
services:
    moveme:
        image: debian:bookworm # or alpine:latest if was built in alpine
        container_name: moveme
        restart: unless-stopped
        ports:
            - "80:80"
        environment:
            DEBUG: "true" # disable debug logging, default is false
            CLEANUP_AFTER: 5m # free up memory after 5 minutes, which is the default
        volumes:
            - ./moveme:/moveme
            - ./routes.csv:/routes.csv
        command: /moveme -c /routes.csv
```

## Routes file example
```csv
gg;https://google.com
yt;https://youtube.com
example;https://example.com
```
Redirects `gg`, `yt` and `example` to `https://google.com`, `https://youtube.com` and `https://example.com` respectively.
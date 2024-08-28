# docker-compose-trigger

## Build

```bash
GOOS=linux GOARCH=amd64 go build -o docker-compose-trigger-linux-amd64 src/main.go
GOOS=linux GOARCH=arm64 go build -o docker-compose-trigger-linux-arm64 src/main.go
```

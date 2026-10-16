# docker-compose-trigger

`docker-compose-trigger` is a small HTTP service for triggering Docker Compose
image pulls and service starts. It looks for each Compose project in a
subdirectory of its current working directory and protects its trigger
endpoints with an API key.

## Requirements

- Go 1.18 or later to build
- Docker with the `docker compose` plugin
- A Compose project (including its Compose file) in the directory for each
  service name

## Configuration

Set these environment variables before starting the program:

| Variable  | Required | Description                                       |
| --------- | -------- | ------------------------------------------------- |
| `ADDRESS` | Yes      | HTTP listen address, for example `:8080`.         |
| `API_KEY` | Yes      | Expected value of the `X-API-Key` request header. |

The process exits at startup if either variable is missing.

## Build and run

Build for Linux amd64 or arm64:

```bash
GOOS=linux GOARCH=amd64 go build -o docker-compose-trigger-linux-amd64 src/main.go
GOOS=linux GOARCH=arm64 go build -o docker-compose-trigger-linux-arm64 src/main.go
```

Then run the binary from the directory that contains the Compose project
directories:

```bash
ADDRESS=127.0.0.1:8080 API_KEY=replace-with-a-secret ./docker-compose-trigger-linux-amd64
```

For example, with a project at `./website/compose.yaml`, the service name
`website` resolves to that `website` directory. The program runs Docker Compose
with that directory as its working directory.

## API

Both trigger endpoints accept **POST** requests and require the configured
`X-API-Key` header:

```bash
curl -X POST -H 'X-API-Key: replace-with-a-secret' \
  http://localhost:8080/trigger/website
```

### `POST /trigger/:service_name`

Pulls the Compose project's images using `docker compose pull`, then starts the
project in the background using `docker compose up -d`.

### `POST /trigger/:service_name/:app_name`

Pulls the named Compose service's image using
`docker compose pull <app_name>`, then runs `docker compose up -d` for the
project. The final `up` command does not specify a service, so it starts the
project rather than only the named service.

For either endpoint, a successful operation responds with `Success` followed
by a newline. An incorrect or missing API key returns HTTP 403 with
`Invalid API-Key`.

## Production deployment with systemd

The example [systemd unit](./docker-compose-trigger.service) listens on port
`9020` on all network interfaces. Put an HTTPS reverse proxy in front of it
and use the host firewall or network controls to restrict direct access to
port `9020` to trusted sources; do not expose the unencrypted HTTP endpoint
publicly. The unit reads its API key from
`/etc/docker-compose-trigger.env`, rather than storing it in the unit file.
Create that file with:

```ini
API_KEY=replace-with-a-new-secret
```

Set the file's owner to `root:root` and permissions to `0600`, then enable the
unit. Replace the placeholder with a newly generated key and configure the
reverse proxy to forward `X-API-Key`.

The unit applies systemd sandboxing and runs as the unprivileged `martas` user.
That user must still be able to access the Docker daemon socket to run Compose.
Membership in the `docker` group effectively grants root-level control of the
host; grant it only if that privilege is acceptable. `ProtectSystem=strict`
makes the filesystem read-only to the service; if a deployment requires local
writes, allow only the specific required path rather than disabling the
restriction.
# bloc-agent

Per-node HTTP daemon for the [Blocstor](https://github.com/blocstor) DRBD storage stack.
Wraps `lvcreate` and `drbdadm`, exposes a JSON REST API consumed by [bloc-manager](https://github.com/blocstor/bloc-manager).

## Install

```sh
go install github.com/blocstor/bloc-agent/cmd/agent@latest
```

Or deploy the systemd unit from `systemd/bloc-agent.service`:

```sh
cp systemd/bloc-agent.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now bloc-agent
```

## Running

```sh
bloc-agent --listen :8080 --log-level info
```

Flags:

| Flag | Default | Description |
|---|---|---|
| `--listen` | `:8080` | TCP address to listen on |
| `--log-level` | `info` | Log level: `debug`, `info`, `warn`, `error` |

## API

All request/response bodies are JSON. Errors return `{"error": "<message>"}`.

| Method | Path | Body | Response |
|---|---|---|---|
| `POST` | `/lv/create` | `{name, vg, size_mb}` | 204 |
| `POST` | `/lv/extend` | `{name, vg, add_mb}` | 204 |
| `POST` | `/lv/remove` | `{name, vg}` | 204 |
| `POST` | `/res/write` | `{name, content}` | 204 — writes `/etc/drbd.d/<name>.res` |
| `POST` | `/res/remove` | `{name}` | 204 — removes `/etc/drbd.d/<name>.res` |
| `POST` | `/drbd/up` | `{resource}` | 204 |
| `POST` | `/drbd/down` | `{resource}` | 204 |
| `POST` | `/drbd/primary` | `{resource}` | 204 |
| `POST` | `/drbd/secondary` | `{resource}` | 204 |
| `POST` | `/drbd/resize` | `{resource}` | 204 |
| `GET` | `/drbd/status?resource=<name>` | — | `{"output": "..."}` 200 |
| `GET` | `/healthz` | — | 200 |

### Example

```sh
# Create a 10 GiB logical volume
curl -X POST http://node1:8080/lv/create \
  -H 'Content-Type: application/json' \
  -d '{"name":"vol0","vg":"data","size_mb":10240}'

# Write a DRBD resource file
curl -X POST http://node1:8080/res/write \
  -H 'Content-Type: application/json' \
  -d '{"name":"r0","content":"resource r0 { ... }"}'

# Bring the resource up
curl -X POST http://node1:8080/drbd/up \
  -H 'Content-Type: application/json' \
  -d '{"resource":"r0"}'

# Check status
curl 'http://node1:8080/drbd/status?resource=r0'
```

## Development

```sh
go build ./...
go vet ./...
go test ./...
```

### Docker

```sh
docker build -t bloc-agent .
docker run --rm -p 8080:8080 bloc-agent
```

## License

Apache 2.0 — see [LICENSE](LICENSE).

# OpenDomus

Open, self-hosted home automation in Go and Vue. OpenDomus turns device readings into
tags you can view, chart, automate, and query from an MCP client.

## Run locally

Requirements: Go 1.24+ and Node.js 22.12+.

```sh
go run ./cmd/opendomus
```

The hub listens on `127.0.0.1:8080` and stores readings, automations, and events in
`opendomus.json`. Build the dashboard and restart the server to serve it from the
same address:

```sh
cd web
npm install
npm run build
cd ..
go run ./cmd/opendomus
```

For frontend development, run the Go service, then use `cd web && npm run dev`.
The Vite development server proxies API requests to the local hub.

Configuration:

| Environment variable | Default | Description |
| --- | --- | --- |
| `OPENDOMUS_ADDR` | `127.0.0.1:8080` | HTTP listen address |
| `OPENDOMUS_DATA` | `opendomus.json` | JSON state file |
| `OPENDOMUS_WEB` | `web/dist` | Built dashboard directory |

The service defaults to loopback because the API is intended for a trusted home
network and does not include user authentication. Only change `OPENDOMUS_ADDR` to
expose it on another interface behind an appropriate network boundary.

## Readings and automations

Submit a reading with a source and one or more named tags:

```sh
curl -X POST http://127.0.0.1:8080/api/readings \
  -H 'Content-Type: application/json' \
  -d '{"source":"living-room","tags":{"temperature":21.5,"humidity":42}}'
```

Provider adapters can send normalized readings to
`POST /api/integrations/{provider}/readings`; the path supplies the reading's
source (for example `shelly`, `sonoff`, or `renault`). The payload uses the same
`timestamp` (optional) and `tags` fields as `/api/readings`. This is the boundary
for device-specific connectors; it accepts readings from integrations without
requiring vendor credentials in the hub.

Useful endpoints:

* `GET /api/tags` — latest value for each tag.
* `GET /api/readings?tag=temperature&limit=100` — reading history.
* `GET` and `POST /api/automations` — list or create a tag threshold rule.
* `GET /api/events` — triggered automation messages.
* `GET /api/health` — service status.

An automation has an `id`, `tag`, `operator` (`eq`, `gt`, `lt`, `gte`, or `lte`),
`value`, and `message`. It is evaluated as matching readings arrive.

## MCP

`POST /mcp` accepts MCP JSON-RPC requests for `initialize`, `tools/list`, and
`tools/call`. The read-only tools are `get_tags` and `query_readings` (optional
`tag` and `limit` arguments). Point an MCP client at `http://127.0.0.1:8080/mcp`.

## Development

```sh
go test ./...
cd web && npm run build
```

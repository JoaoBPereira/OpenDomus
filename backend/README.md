# OpenDomus backend (Go)

The server: it polls the devices, stores every reading in PostgreSQL, runs the automations
and serves the REST API used by `../frontend` and `../backoffice`, plus an MCP server at `/mcp`.

## Features

- Device polling for Shelly (1, EM, Plug S, Plus/Gen3 Plug S, Plug M Gen3), NOUS A1T, SNZB-02D
  (MQTT), EcoForest, Rohan Air (Tuya), Renault (MyRenault) and Yeelight.
- Tags with float, boolean and location values and full history (TimescaleDB).
- Automations as a node graph: timer, sunrise/sunset and tag-value triggers; and/or/condition/
  delayed-check gates; device, HTTP and WhatsApp actions.
- JWT login, `admin` user created from `ADMIN_PASSWORD`, API-key endpoints for sensors and phones
  (`/saveTagValue`, `/location`).
- Weather from IPMA, daily maintenance job, WhatsApp alerts via callmebot.
- MCP tools for AI assistants: `list_devices`, `list_tags`, `get_tag_latest_value`,
  `get_tag_history`, `get_tag_history_between_dates`.

## Commands

```sh
go run ./cmd/od              # dev (reads .env from the current directory)
go run ./cmd/od --debug      # keep tag values in memory only (no tag value writes)
go build -o od ./cmd/od     # single binary
```

Docker, from the repository root: `docker build -f Dockerfile.go -t od-app:go .` builds the same image
layout as `Dockerfile` (nginx + both frontends + the backend on port 4000).

Required variables: `JWT_SECRET`, `DB_HOST`, `DB_NAME`, `DB_USER` (see `.env.example`). `ADMIN_PASSWORD`
is read once, to create the `admin` user when it does not exist.

## Layout

```
cmd/od/            main.go (config, database, signals), app.go (wires the managers and servers)
internal/
  config/           Config, loaded once from the environment
  postgres/         connection pool (Query, QueryInto, Exec) and the schema migrations
  logger/           slog handler: console, daily od-*.log file, public.log table
  httpclient/       shared http.Client (timeouts come from the context)
  parse/            loosely typed values (numbers as strings, dates) into Go values
  schedule/         cron jobs (robfig/cron, 5 or 6 fields)
  models/           plain data: Tag, TagValue*, DeviceFields, Automation, pagination
  devices/          one type per device: Shelly*, NousA1T, EcoForest, RohanAir (Tuya), SNZB02D (MQTT), Renault, YeeDevice
  renault/          MyRenault API client
  managers/         TagsManager, DevicesManager, UsersManager, AutomationsManager, TasksManager, WeatherIPMAManager
  rest/             http.Server and the middlewares; rest/routes holds the handlers (Handlers struct) and the JWT Auth
  mcp/              MCP server mounted at /mcp
```

Dependencies are passed in constructors: `main` builds the config, the database and the managers,
and hands each package what it needs. The devices see the TagsManager through the small
`devices.TagStore` interface. Every database call takes a `context.Context` (the request context in
the handlers, a bounded one in the jobs) and errors are returned to the caller; the handlers map
`managers.ErrNotFound` to 404 and everything else to 500.

Logging is `log/slog` with structured fields: `slog.Info("tags loaded", "count", n)`. The handler in
`internal/logger` writes each record to the console, to a daily rotating file (kept 14 days) and to
the `public.log` table.

The REST routes are registered in lowercase too, because the frontend calls `/powerconsumption/...`
for the `/powerConsumption/...` route (Express matched any case).

## Notes from the port

This is a port of the original Node/TypeScript backend, which is no longer in the repository.

### Differences from the TypeScript version

Fixed on purpose:

- `GET /nearestTagValueFloat` works (the TypeScript SQL was rejected by PostgreSQL, so it always answered `0`).
- The startup database check tested `rowCount > 1`, which always tried to `CREATE DATABASE`.
- `POST /restart` keeps the `--debug` flag and restarts only the managers; the HTTP server keeps running.
- Invalid timestamps and non-numeric values in `POST /saveTagValue` are rejected (`{false}`) instead of
  storing `NaN` or failing later.
- `GET /devices`, `POST /devices` and `PUT /devices/:id` return the rows of the device table; they no
  longer build live devices (each call used to start a new polling job that was never stopped).
- A database error while loading the tags stops the startup instead of running with no tags.
- `GET /powerConsumption` looks up the device type of the tag again, so the `NOUS A1T` counter rule
  applies (the port had left it as a TODO).
- The `users.role` column and the admin role are a schema migration (v2) instead of running on every start.

Kept on purpose, to behave the same:

- Scheduled HTTP commands are listed and updated but never fired (`schedule()` was never called).
- The Shelly Plug S reports a relay that is off as unknown, so the P1/P2 task never turns that plug back on.
- Optional fields missing from `POST/PUT /devices` and `/tags` are stored as empty strings / 0.
- Devices are started whatever their `enabled` column says.

Other notes:

- **Rohan Air (Tuya)**: `devices/tuya.go` is a port of the `tuyapi` 3.3 protocol and was not tested against
  the real device yet. It discovers the device IP by UDP; putting the IP in the device `url` skips the discovery.
- **Log files** rotate daily and are deleted after 14 days; the 20 MB cap and gzip of winston are not implemented.
- Jobs run in goroutines, so `Tag.LastTagValue`, the tag list of `TagsManager` and the device list of
  `DevicesManager` are protected by mutexes: use `tag.LastValue()` / `tag.SetLastValue()`.
- Every query has a 30 second timeout (`postgres.QueryTimeout`).

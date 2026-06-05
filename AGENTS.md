# smart-chapa

## Run

```bash
go run ./cmd        # starts on :8080 (set PORT in .env)
```

## Build & verify

```bash
go build ./...      # no linter/typecheck config — just compile
```

## DB

- SQLite via `modernc.org/sqlite` (pure Go, no CGO).
- Schema is inline in `internal/db/db.go` — no migration files.
- DB file `smart_chapa.db` is auto-created, gitignored.
- Delete it to reset: `Remove-Item smart_chapa.db -ErrorAction SilentlyContinue`

## Env (`.env`)

```
PORT=8080
MQTT_HOST=<hivemq-cloud-host>
MQTT_USER=<user>
MQTT_PASSWORD=<pass>
```

## Project layout

```
cmd/main.go               # entrypoint — wires DB, MQTT, routes
internal/db/db.go         # DB init + CREATE TABLE statements
internal/models/models.go # Go structs (Device, DoorEvent, LightEvent, PendingCommand)
internal/handlers/*.go    # HTTP handlers — struct with *sql.DB + optional mqtt.Client
```

## Architecture

- HTTP router: `chi/v5`
- Handler pattern: struct with injected `*sql.DB` (and `mqtt.Client` if needed), constructor `New*`, methods `func(w, r)`.
- MQTT: `eclipse/paho.mqtt.golang` v1, TLS to HiveMQ Cloud.
- New feature pattern: `db table` → `model` → `handler` (subscribe + endpoints) → `main.go routes`.
- Adding a new handler: create file in `internal/handlers/`, add table + model, register in `cmd/main.go`.

## MQTT topics

| Topic | Direction | Payload |
|-------|-----------|---------|
| `door/cmd` | publish | `"open"` |
| `lights/cmd` | publish | `"turn_on"` / `"turn_off"` |
| `lights/status` | subscribe | `"relay_on"` / `"relay_off"` (from ESP32, saved to DB) |

## API endpoints

All under `/api`.

| Method | Path | Handler |
|--------|------|---------|
| POST | `/door/open` | publishes MQTT |
| POST | `/door/confirm` | inserts `door_events` |
| GET | `/door/events` | last 50 door events |
| POST | `/devices` | create device with random token |
| GET | `/devices` | list devices |
| POST | `/lights/on` | publishes `turn_on` to `lights/cmd` |
| POST | `/lights/off` | publishes `turn_off` to `lights/cmd` |
| GET | `/lights/events` | last 50 light events from `lights/status` subscription |

## Key facts

- No tests, no CI, no linter, no formatter config.
- No auth middleware — tokens in `devices` table exist but are unused.
- `pending_commands` table exists but no code reads/writes it.
- Responses are manual `json.NewEncoder(w).Encode(...)` — no response helpers.
- All JSON payloads are plain `struct` fields decoded inline — no shared DTOs.
- MQTT subscription happens in the handler constructor (`NewLightHandler`).

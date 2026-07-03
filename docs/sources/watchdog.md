# Watchdog

Esp32 watchdog and go consumer that tracks uptime events (up and down) of both devices (the home lab and the esp32 watchdog)

## Topics mqtt

`events/uptime/lab` and `events/uptime/watchdog` — derived uptime events.

```json
{ "state": "up", "timestamp": "2026-05-31T04:01:00Z" }
```

`state` is `up` or `down`; `timestamp` is RFC 3339 UTC.

## Tables

Raw events land in the shared `raw.mqtt_events` table (see `internal.md`).

### Staging — `staging.watchdog_uptime`

| Column | Type | Description |
| ------ | ---- | ----------- |
| `event_id` | `BIGINT` | From the raw event id. |
| `device` | `ENUM('lab','watchdog')` | Device the event belongs to. |
| `state` | `ENUM('up','down')` | State it switches to. |
| `event_time` | `TIMESTAMPTZ` | When the switch occurred (UTC). |

### Marts

#### `marts.uptime_windows`

| Column | Type | Description |
| ------ | ---- | ----------- |
| `device` | `ENUM('lab','watchdog')` | Device the window is for. |
| `state` | `ENUM('up','down')` | State of the window. |
| `start_time` | `TIMESTAMPTZ` | When the window starts. |
| `end_time` | `TIMESTAMPTZ` | When the window ends (nullable). |

**PK:** (`device`, `start_time`)

#### `marts.uptime_aggregations`

| Column | Type | Description |
| ------ | ---- | ----------- |
| `device` | `ENUM('lab','watchdog')` | Device the aggregation is for. |
| `uptime` | `float` | Percentage of uptime. |
| `time` | `ENUM('month','3 months','year','all')` | Time range of the aggregation. |

**PK:** (`device`, `time`)

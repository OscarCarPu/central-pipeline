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

Every column below is materialised as `TEXT` or `TIMESTAMPTZ` — dbt has no enum types, so the value sets are enforced by `accepted_values` tests instead of by the database. Consumers should treat them as text with a closed value set.

### Staging — `staging.watchdog_uptime`

| Column | Type | Description |
| ------ | ---- | ----------- |
| `event_id` | `BIGINT` | From the raw event id. |
| `device` | `TEXT` | Device the event belongs to. |
| `state` | `TEXT` | State it switches to. |
| `event_time` | `TIMESTAMPTZ` | When the switch occurred (UTC). |

### Marts

#### `marts.uptime_windows`

| Column | Type | Description |
| ------ | ---- | ----------- |
| `device` | `TEXT` | Device the window is for. |
| `state` | `TEXT` | State of the window. |
| `start_time` | `TIMESTAMPTZ` | When the window starts. |
| `end_time` | `TIMESTAMPTZ` | When the window ends (nullable). |

**PK:** (`device`, `start_time`)

One window per state change: consecutive events of the same state collapse into the running window. `end_time` is null for the newest window of each device — the state it is in now. Exactly one open window per device.

#### `marts.uptime_aggregations`

| Column | Type | Description |
| ------ | ---- | ----------- |
| `device` | `TEXT` | Device the aggregation is for. |
| `uptime` | `float` | Percentage of uptime, 0-100 with two decimals. |
| `time` | `TEXT` | Time range of the aggregation. |
| `range_start` | `TIMESTAMPTZ` | Start of the range the percentage covers. |
| `range_end` | `TIMESTAMPTZ` | End of the range — the dbt run time. |

**PK:** (`device`, `time`)

Four precomputed ranges, anchored to the run time, so a consumer gets a percentage per lookback with one indexed read and no date math. Arbitrary ranges are not served here — read `uptime_windows` and clip. `range_end` is also the freshness marker: the numbers are as stale as the last dbt run.

Every range is floored at the device's first event, so a device with a week of history reports its real week under `year` rather than 2% uptime. Windows straddling a range boundary are clipped, not dropped. The open window counts up to `range_end`, which means the latest known state is assumed to persist — a device that dies without a `down` event keeps reading as up until its peer reports it.

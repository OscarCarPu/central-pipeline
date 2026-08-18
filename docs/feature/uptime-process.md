# Feature: uptime-process

Consumes `events/uptime/lab` and `events/uptime/watchdog` from MQTT, persists raw events to Postgres, and transforms them into uptime windows via dbt for gv-api to serve.

Disk queue and retry logic live here, not in the source projects.

## Simulator

`cmd/simulator` publishes the documented uptime payloads as `central-pipeline-sim`, so this feature can be tested without the real devices.

| Mode | Purpose |
| ---- | ------- |
| `live` | Alternating events every `-interval`, timestamped now. |
| `backfill` | `-since` of history in one burst, for the mart aggregation ranges. |
| `once` | A single `-device`/`-state` event. |

`-seed` makes a run reproducible. `-duplicates`, `-skew` and `-malformed` generate the cases the ingestion contract claims to absorb: at-least-once redelivery, out-of-order timestamps and payloads that are not valid JSON.

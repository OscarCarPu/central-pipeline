# central-pipeline

Central IoT data pipeline for the home lab. Subscribes to all MQTT topics, writes raw events to Postgres, transforms them with dbt, and feeds gv-api.

Each IoT project (mutual-watchdog, plant sensors, animal feeder, etc.) is a producer — it publishes raw telemetry and derived events to MQTT. central-pipeline owns everything from MQTT onward: ingestion, persistence, and transformation. gv-api reads from the clean models.

Follows the [medallion architecture](docs/internal.md) (bronze → silver → gold) with Postgres as the store and dbt handling both silver and gold layers.

```
MQTT (all topics)
  → Go consumer → raw schema (bronze)
  → dbt staging (silver)
  → dbt marts (gold)
  → gv-api
```

## Sources

| Source | MQTT topics | Data model |
|---|---|---|
| mutual-watchdog | `devices/esp32/ping`, `events/uptime/#` | uptime windows |

## Features

- [uptime-process](docs/feature/uptime-process.md)

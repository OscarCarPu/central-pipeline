# central-pipeline

Central IoT data pipeline for the home lab. Subscribes to the MQTT topics of each registered source, writes raw events to Postgres, transforms them with dbt, and feeds gv-api.

Each IoT project (mutual-watchdog, plant sensors, animal feeder, etc.) is a producer — it publishes raw telemetry and derived events to MQTT. central-pipeline owns everything from MQTT onward: ingestion, persistence, and transformation. gv-api reads from the clean models.

Follows the [medallion architecture](docs/internal.md) (bronze → silver → gold) with Postgres as the store and dbt handling both silver and gold layers.

```
MQTT (registered topics)
  → Go consumer → raw (bronze)
  → dbt staging (silver)
  → dbt marts (gold)
  → gv-api
```

## Local development

Requires Docker with Compose, and the [mqtt-broker](https://github.com/OscarCarPu/mqtt-broker) project running — this compose file starts Postgres only.

```
cp .env.example .env   # set your credentials
make up                # start Postgres, wait until healthy
```

Postgres listens on `localhost:5432`. On first start the SQL in `initdb/` runs automatically to create the schemas.

The consumer authenticates against the broker as `central-pipeline`, which needs both a password (`make create-password`) and an entry in the broker's ACL — a user missing from the ACL connects fine and then silently receives nothing.

| Target | Description |
|---|---|
| `make up` | Start and wait until healthy |
| `make down` | Stop containers (data kept) |
| `make restart` | Rebuild from scratch — **drops the database volume** |

## Sources

| Source | MQTT topics | Data model |
|---|---|---|
| mutual-watchdog | `watchdog/ping`, `events/uptime/#` | uptime windows |


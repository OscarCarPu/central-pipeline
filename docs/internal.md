# Internal architecture

## Medallion layers

Three layers, one Postgres schema each.

| Layer | Schema | Owner | Purpose |
|---|---|---|---|
| Bronze | `raw` | Go consumer | Immutable. Exact payload from MQTT, no transformation. |
| Silver | `staging` | dbt | Cast types, rename fields, drop nulls. One model per source. |
| Gold | `marts` | dbt | Aggregated business logic. What gv-api reads. |

```
Go consumer → raw.mqtt_events (bronze)
                  ↓
            dbt staging (silver)  — one per source: watchdog_uptime, ...
                  ↓
            dbt marts (gold)      — uptime_windows, ...
                  ↓
            gv-api
```

Bronze is never modified after insert — fix a buggy model and rerun dbt; raw is always there to replay from.

Per-source staging and mart models live under [`docs/sources/`](sources/).

## Raw table — `raw.mqtt_events`

One generic table for every source. The consumer inserts each MQTT message untouched — no parsing.

| Column | Type | Description |
| ------ | ---- | ----------- |
| `id` | `BIGINT` (autoincrement) | Surrogate key, ingestion order. |
| `topic` | `TEXT` | Full MQTT topic. Carries the source/device. |
| `payload` | `JSONB` | Exact MQTT payload, unparsed. |
| `received_at` | `TIMESTAMPTZ` | When the consumer ingested the message. |


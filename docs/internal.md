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

The table is generic, the subscription is not: the topic list lives in the consumer, alongside the client id and QoS. Those change with features rather than with the environment, so they are code, not configuration. Only the broker address and credentials come from `.env`.

## Ingestion contract

The consumer connects with a fixed client id, a persistent session (clean session off) and subscribes at QoS 1. All three are required together — drop any one and the broker stops queueing messages while the consumer is down.

That gives at-least-once delivery, so **duplicates are possible**: a redelivery after a lost ack inserts the same event twice. Bronze keeps them, since it records what actually arrived. Staging deduplicates — `staging.watchdog_uptime` on (`device`, `state`, `event_time`), keeping the lowest `event_id`.

The broker queues at most `max_queued_messages` (1000, its default) per offline session. A long enough outage drops the oldest messages beyond that.


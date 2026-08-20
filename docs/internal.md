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
            dbt marts (gold)      — uptime_windows, uptime_aggregations, ...
                  ↓
            gv-api
```

Bronze is never modified after insert — fix a buggy model and rerun dbt; raw is always there to replay from.

Per-source staging and mart models live under [`docs/sources/`](sources/).

## Runtime shape

Bronze is streaming, silver and gold are batch. The consumer is a long-running service, not a scheduled job — the at-least-once guarantee below depends on holding one persistent MQTT session, and a process that connected and disconnected on a timer would make the broker's offline queue the normal delivery path instead of the outage path. dbt is the opposite: a batch job over whatever bronze holds, so it is scheduled rather than resident.

Both run as containers. The consumer is a compose service with a restart policy; dbt sits behind the `tools` profile and is invoked per run, on a timer once deployed.

One dbt run rebuilds every mart, so all of them carry the same freshness timestamp — consumers treat "the pipeline" as one clock, and gv-api derives a single staleness flag from it. Adding a source that needs its own cadence (a sensor read every 15 minutes against a daily rebuild) breaks that assumption silently: the shared threshold becomes meaningless rather than wrong. Split the runs only together with telling consumers, so freshness becomes per-mart on both sides at once.

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

The consumer subscribes to the topics in `internal/topics` and inserts each message into `raw.mqtt_events` untouched. `payload` is `JSONB`, which rejects anything that is not valid JSON, so a payload that is not gets stored as a JSON string instead of being dropped — `not json` becomes `"not json"`. Staging models filter with `jsonb_typeof(payload) = 'object'`.

The consumer connects with a fixed client id, a persistent session (clean session off) and subscribes at QoS 1. All three are required together — drop any one and the broker stops queueing messages while the consumer is down.

That covers the subscribe side only. Delivery happens at `min(publish QoS, subscribe QoS)`, and mutual-watchdog currently publishes uptime events at QoS 0, non-retained, so the broker queues nothing for this consumer while it is down: those events are lost outright, not delayed. The guarantee below therefore holds against a redelivery after a lost ack, but not against pipeline downtime — and a lost event is silent, since a missing `down` closes no window and simply overstates uptime. Closing it needs mutual-watchdog publishing at QoS 1; until then, treat consumer downtime as data loss and prefer a backfill over assuming raw is complete.

That gives at-least-once delivery, so **duplicates are possible**: a redelivery after a lost ack inserts the same event twice. Bronze keeps them, since it records what actually arrived. Staging deduplicates — `staging.watchdog_uptime` on (`device`, `state`, `event_time`), keeping the lowest `event_id`.

The consumer acks a message only after its row is committed, so a failed insert leaves it unacked and the broker redelivers it when the session resumes. That persistent session is the queue for now; a local disk spool is only needed to survive the broker's own ceiling.

The broker queues at most `max_queued_messages` (1000, its default) per offline session. A long enough outage drops the oldest messages beyond that.


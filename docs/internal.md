# Internal architecture

## Medallion layers

Data flows through three layers. Each layer is a Postgres schema.

| Layer | Schema | Owner | Purpose |
|---|---|---|---|
| Bronze | `raw` | Go consumer | Immutable. Exact payload from MQTT, no transformation. |
| Silver | `staging` | dbt | Cast types, rename fields, drop nulls. One model per source. |
| Gold | `marts` | dbt | Aggregated business logic. What gv-api reads. |

```
Go consumer → raw (bronze)
                  ↓
            dbt staging models (silver)  — stg_uptime_events, stg_temperature_readings, ...
                  ↓
            dbt mart models (gold)       — mart_uptime_windows, mart_daily_temperature, ...
                  ↓
            gv-api
```

Bronze is never modified after insert. If a dbt model has a bug, fix the SQL and rerun dbt — the raw data is always there to replay from.

## dbt model folders

```
models/
  staging/    ← silver
  marts/      ← gold
```

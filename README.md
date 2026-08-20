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

Requires Docker with Compose, and the [mqtt-broker](https://github.com/OscarCarPu/mqtt-broker) project running — this compose file has no broker.

```
cp .env.example .env   # set your credentials
make up                # start Postgres, wait until healthy
```

Every variable in `.env.example` is required. Compose fails fast rather than starting a service with a blank credential, so an unset `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, `MQTT_PORT`, `MQTT_USERNAME` or `MQTT_PASSWORD` aborts the command.

Postgres publishes `POSTGRES_PORT` (54321 by default, so it does not collide with a plain Postgres on 5432 — the server runs this database alongside gv's) — change it if another project already owns that port. Containers reach the database at `db:5432` regardless, since the published port and the in-network one are independent. On first start the SQL in `initdb/` runs automatically to create the schemas.

The consumer authenticates against the broker as `central-pipeline`, which needs both a password (`make create-password`) and an entry in the broker's ACL — a user missing from the ACL connects fine and then silently receives nothing.

`make simulate` publishes fake uptime events as `central-pipeline-sim`, so the pipeline can be exercised without the real devices. See [cmd/simulator](cmd/simulator).

### Running the consumer

Two ways, same binary. `make consume` runs it on the host against `MQTT_HOST` — the quickest loop while editing Go. `make consumer-up` builds the image and runs it as an always-on compose service, which is how it is deployed; `make consumer-logs` follows it.

In a container `localhost` is the container itself, so the broker needs a different address there: `MQTT_HOST_DOCKER`, defaulting to `host.docker.internal` to reach a broker published on the host. Point it at the broker's service name instead if you attach this project to the broker's compose network.

The consumer exits on a failed initial connect and the restart policy brings it back, so a broker that is not up yet shows as a restart loop logging `connection refused` — it recovers on its own once the broker answers.

`make up` deliberately starts Postgres only. The consumer is left out so the dev database comes up whether or not a broker is running.

### Running dbt

dbt runs as a container too, under the `tools` compose profile so it stays out of `make up` and `make down`. `make dbt-run` executes `dbt build` — models and tests together, so a failing test stops the run instead of publishing bad rows to `marts`. Credentials come from the same `.env`; `dbt/profiles.yml` reads them via `env_var()`.

| Target | Description |
|---|---|
| `make up` | Start Postgres and wait until healthy |
| `make down` | Stop containers (data kept) |
| `make restart` | Rebuild from scratch — **drops the database volume** |
| `make test` | Run the Go tests |
| `make simulate` | Publish simulated uptime events (`ARGS="-mode=backfill"`) |
| `make consume` | Run the consumer on the host: MQTT → `raw.mqtt_events` |
| `make consumer-up` | Build and start the consumer as a compose service |
| `make consumer-logs` | Follow the consumer container logs |
| `make dbt-run` | Run `dbt build` — staging and marts, with tests |
| `make test-integration` | Run the tests that need Postgres up |
| `make db` | Open a pgcli session against the database |

## Sources

| Source | MQTT topics | Data model |
|---|---|---|
| mutual-watchdog | `watchdog/ping`, `events/uptime/#` | uptime windows |


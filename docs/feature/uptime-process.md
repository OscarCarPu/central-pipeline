# Feature: uptime-process

**Status: built, not deployed.** Bronze, silver and gold are implemented and tested locally: `raw.mqtt_events` (Go consumer), `staging.watchdog_uptime`, `marts.uptime_windows`, `marts.uptime_aggregations`. gv-api reads the two marts directly over Postgres — see [sources/watchdog.md](../sources/watchdog.md) for the column contracts. The gv side is fully deployed (api and web) and degrades cleanly while this stack is absent; deploying it is the remaining step, written up under [Deployment](#deployment) and deliberately deferred.

Consumes `events/uptime/lab` and `events/uptime/watchdog` from MQTT, persists raw events to Postgres, and transforms them into uptime windows via dbt for gv-api to serve.

Retry logic lives here, not in the source projects. The queue is the broker's persistent session rather than a local disk spool — see the ingestion contract in [internal.md](../internal.md).

## Simulator

`cmd/simulator` publishes the documented uptime payloads as `central-pipeline-sim`, so this feature can be tested without the real devices.

| Mode | Purpose |
| ---- | ------- |
| `live` | Alternating events every `-interval`, timestamped now. |
| `backfill` | `-since` of history in one burst, for the mart aggregation ranges. |
| `once` | A single `-device`/`-state` event, at `-at` (RFC 3339) or now. |

`-at` places a `once` event at a chosen instant rather than now, which is what constructs a "device installed at X" state — two `once` events after a volume wipe give a pair of devices with one clean window each and no prior history. `-seed` makes a run reproducible. `-duplicates`, `-skew` and `-malformed` generate the cases the ingestion contract claims to absorb: at-least-once redelivery, out-of-order timestamps and payloads that are not valid JSON.

## Deployment

Deferred, not done. gv-api and gv-web are both live on `ssh.lab-ocp.com`: the uptime endpoints are registered and answering 503 (`PIPELINE_DATABASE_URL not set`), and the Uptime tab exists in the prod UI showing its empty state — not a 404 — until this stack lands there. That degradation is by design — nothing on the gv side needs changing when it does.

Already on that host: `gv_api`, `gv_db`, `gv-web`, `mosquitto` + `mosquitto-ui` on 1883/9001, `mutual-watchdog-consumer`, gitea, minecraft, seafile. Missing: this whole stack — no `db_central_pipeline`, no consumer, no `/home/ocp/docker/central-pipeline`.

### Steps

1. Clone to `/home/ocp/docker/central-pipeline`, alongside the existing `mqtt-broker` and `mutual-watchdog`.
2. `cp .env.example .env` and fill it in. Two values differ from local:
   - `POSTGRES_PORT=54321` — the default, and it must stay off 5432, which `gv_db` already owns.
   - `MQTT_HOST_DOCKER=mosquitto` if the consumer joins the broker's compose network, otherwise leave the `host.docker.internal` default and reach the broker on the published port.
3. Give the `central-pipeline` broker user a password (`make create-password` in mqtt-broker) **and** an ACL entry. Missing from the ACL, it connects fine and then silently receives nothing.
4. `make up` — Postgres only; `initdb/` creates the three schemas on first start.
5. `make consumer-up`, then `make consumer-logs`. The line to look for is `consuming client_id=... topics=[watchdog/ping events/uptime/#]` — both, since `internal/topics` subscribes to the ping topic too. `watchdog/ping` appearing there proves only that the subscription was accepted; nothing publishes to it yet, so it stays empty.
6. `make dbt-deps` once, then `make dbt-run`.
7. Create the read-only role gv-api reads with, and grant it so the grant survives dbt:
   ```sql
   CREATE ROLE gv_api_ro LOGIN PASSWORD '...';
   GRANT USAGE ON SCHEMA marts TO gv_api_ro;
   GRANT SELECT ON ALL TABLES IN SCHEMA marts TO gv_api_ro;
   ALTER DEFAULT PRIVILEGES IN SCHEMA marts GRANT SELECT ON TABLES TO gv_api_ro;
   ```
   The `ALTER DEFAULT PRIVILEGES` line is the important one: dbt drops and recreates both mart tables on every run, so a per-table grant alone stops working at the next run. gv-api retries `42P01`, which would make that read as intermittent rather than broken.

   Run these as the same role dbt connects with — `POSTGRES_USER` from `.env`. Default privileges attach to the role that creates the object, not to the schema, so the same statement run as a different superuser applies to that superuser's future tables and silently does nothing for dbt's. Add `FOR ROLE <dbt user>` if you have to run it as someone else. The failure looks identical to forgetting the line entirely, one dbt run later.
8. Add `PIPELINE_DATABASE_URL=postgresql://gv_api_ro:...@host.docker.internal:54321/<db>?sslmode=disable` to `/home/ocp/docker/gv/gv-api/.env` and restart `gv_api`. Its container already carries `extra_hosts: host.docker.internal:host-gateway` — verified live on that host, where it resolves to `172.17.0.1`:
   ```sh
   docker exec gv_api getent hosts host.docker.internal
   ```
   If that prints nothing, the container predates the entry and needs recreating (`docker compose up -d --force-recreate gv-api`) rather than restarting; a missing mapping fails as a connection error and reads like a wrong DSN. Keep the two stacks on separate networks — gv-api could join this one and use `db:5432`, but that couples their lifecycles for no gain.

9. Confirm it from the gv side before believing it. The startup warning must be gone, and the endpoint must answer with data:
   ```sh
   docker logs gv_api 2>&1 | grep -i pipeline   # no "PIPELINE_DATABASE_URL not set"
   curl -s -H "Authorization: Bearer $TOKEN" https://gv-api.lab-ocp.com/domotics/uptime  # make auth (in gv-api) prints a token
   ```
   A 200 carrying `computed_at` means the whole path works. A 503 means gv-api never got the DSN (env not reloaded — restart, do not just `docker restart` a container whose env_file changed); a 500 means it connected and the query failed, which at this point is the grant from step 7.
10. Schedule `make dbt-run` (cron or a systemd timer). Nothing runs it yet, and `range_end` is only as fresh as the last run.

11. Match gv-api's staleness threshold to whatever cadence step 10 uses. `PIPELINE_STALE_AFTER_MS` defaults to 2h, so a daily dbt run leaves every read flagged `stale: true` and the Uptime tab permanently captioned "not live" — technically true, useless as a signal. Set it to a little over the interval (hourly run → ~90min; daily → ~26h) in `/home/ocp/docker/gv/gv-api/.env`.

### Two ways this comes undone later

**`make restart` takes the role with it.** It runs `docker compose down -v`, and the role from step 7 lives in that volume — `initdb/` only creates the schemas and `raw.mqtt_events`. So a restart meant to reset data also deletes gv-api's credentials, and gv drops back to 503 (or 500) with nothing in this stack looking wrong. Either redo step 7 after any volume wipe, or move the role into `initdb/03_gv_api_ro.sh` so it comes back with everything else — a shell script rather than `.sql`, since plain SQL cannot read the environment and the password should not be committed. The entrypoint runs `.sh` and `.sql` alike, with the compose `environment:` available, so the script reads `GV_API_RO_PASSWORD` and skips itself when that is unset, leaving local development unaffected. Not written yet: it is deploy-time work, and it only pays off once a role exists to recreate. The backfill is in that volume too.

**Backing out is one line.** Remove `PIPELINE_DATABASE_URL` from gv-api's `.env` and restart it: the domain returns to its designed 503 and the rest of gv is untouched. Worth knowing before touching prod — nothing here needs a gv-side rollback or redeploy.

### What to expect on the first day

The real producer needs no adapter — mutual-watchdog publishes exactly the documented topics and payload, so prod gets real `lab` history from deploy day forward, with `watchdog` sparse while the ESP32 is down. The simulator is not needed there.

But publishes are QoS 0 and non-retained (see the ingestion contract in [internal.md](../internal.md)), so there is no history before deploy and no current-state message on connect. Until the first real transition, `staging.watchdog_uptime` is empty and gv-api reports `state: "unknown"` with no ranges. On a stable lab that can be days. That is the correct path, not a wiring fault — verify the consumer's subscription line rather than assuming the DSN is wrong.

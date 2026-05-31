# Feature: uptime-process

Consumes `events/watchdog/*` and `events/lab/*` from MQTT, persists raw events to Postgres, and transforms them into uptime windows via dbt for gv-api to serve.

Disk queue and retry logic live here, not in the source projects.

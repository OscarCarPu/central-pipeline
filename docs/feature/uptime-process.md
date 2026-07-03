# Feature: uptime-process

Consumes `events/uptime/lab` and `events/uptime/watchdog` from MQTT, persists raw events to Postgres, and transforms them into uptime windows via dbt for gv-api to serve.

Disk queue and retry logic live here, not in the source projects.

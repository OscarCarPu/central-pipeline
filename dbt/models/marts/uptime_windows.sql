{# One row per state change. Staging dedups on (device, state, event_time), so a
   device can still hold two different states at one instant — DISTINCT ON keeps
   the lowest event_id, matching staging's tie-break, and guarantees the PK. #}
WITH events AS (
  SELECT DISTINCT ON (device, event_time)
    device,
    state,
    event_time
  FROM {{ ref('watchdog_uptime') }}
  ORDER BY device, event_time, event_id
),

transitions AS (
  SELECT
    device,
    state,
    event_time
  FROM (
    SELECT
      *,
      lag(state) OVER (PARTITION BY device ORDER BY event_time) AS prev_state
    FROM events
  ) AS flagged
  WHERE prev_state IS null OR prev_state <> state
)

SELECT
  device,
  state,
  event_time AS start_time,
  lead(event_time) OVER (PARTITION BY device ORDER BY event_time) AS end_time
FROM transitions

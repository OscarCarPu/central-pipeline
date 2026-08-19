WITH source AS (
  SELECT
    id AS event_id,
    split_part(topic, '/', 3) AS device,
    payload ->> 'state' AS state,
    CASE
      WHEN payload ->> 'timestamp' ~ '^\d{4}-\d{2}-\d{2}[T ]'
        THEN (payload ->> 'timestamp')::timestamptz
    END AS event_time
  FROM {{ source('raw','mqtt_events') }}
  WHERE
    topic LIKE 'events/uptime/%'
    AND jsonb_typeof(payload) = 'object'
),

valid AS (
  SELECT *
  FROM source
  WHERE
    device IN ('lab', 'watchdog')
    AND state IN ('up', 'down')
    AND event_time IS NOT null
),

deduplicated AS (
  SELECT
    *,
    row_number() OVER (
      PARTITION BY device, state, event_time
      ORDER BY event_id
    ) AS row_num
  FROM valid
)

SELECT
  event_id,
  device,
  state,
  event_time
FROM deduplicated
WHERE row_num = 1

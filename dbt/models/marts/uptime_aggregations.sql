{# Uptime percentage per device and range, anchored to the run time. Ranges are
   floored at the device's first event so a young device is not punished for the
   time before it existed. #}
WITH ranges AS (
  SELECT *
  FROM (VALUES
    ('month', interval '1 month'),
    ('3 months', interval '3 months'),
    ('year', interval '1 year'),
    ('all', null::interval)
  ) AS r (range_name, lookback)
),

first_seen AS (
  SELECT
    device,
    min(start_time) AS first_time
  FROM {{ ref('uptime_windows') }}
  GROUP BY device
),

bounds AS (
  SELECT
    f.device,
    r.range_name,
    greatest(f.first_time, coalesce(now() - r.lookback, f.first_time)) AS range_start,
    now() AS range_end
  FROM first_seen AS f
  CROSS JOIN ranges AS r
),

{# The join predicates overlap-filter; least/greatest clip a straddling window
   to the range instead of counting or dropping it whole. #}
clipped AS (
  SELECT
    b.device,
    b.range_name,
    b.range_start,
    b.range_end,
    b.range_end - b.range_start AS total,
    sum(
      least(coalesce(w.end_time, b.range_end), b.range_end)
      - greatest(w.start_time, b.range_start)
    ) FILTER (WHERE w.state = 'up') AS up
  FROM bounds AS b
  LEFT JOIN {{ ref('uptime_windows') }} AS w
    ON
      b.device = w.device
      AND w.start_time < b.range_end
      AND coalesce(w.end_time, b.range_end) > b.range_start
  GROUP BY b.device, b.range_name, b.range_start, b.range_end
)

SELECT
  device,
  range_name AS "time",
  range_start,
  range_end,
  round(
    100 * extract(epoch FROM coalesce(up, interval '0'))
    / nullif(extract(epoch FROM total), 0), 2
  )::float AS uptime
FROM clipped

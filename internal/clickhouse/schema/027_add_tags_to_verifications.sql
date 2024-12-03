-- +goose up
ALTER TABLE verifications.raw_key_verifications_v1
ADD COLUMN tags Array(String) DEFAULT [];

ALTER TABLE verifications.key_verifications_per_hour_v1
ADD COLUMN tags Array(String) DEFAULT [];

ALTER TABLE verifications.key_verifications_per_day_v1
ADD COLUMN tagsArr Array(String) DEFAULT [];

ALTER TABLE verifications.key_verifications_per_month_v1
ADD COLUMN tags Array(String) DEFAULT [];

ALTER TABLE verifications.key_verifications_per_hour_mv_v1
MODIFY QUERY
SELECT
	workspace_id,
	key_space_id,
	identity_id,
	key_id,
	outcome,
	count(*) AS COUNT,
	toStartOfHour(fromUnixTimestamp64Milli (time)) AS time,
	tags
FROM
	verifications.raw_key_verifications_v1
GROUP BY
	workspace_id,
	key_space_id,
	identity_id,
	key_id,
	outcome,
	time,
	tags
	;



ALTER TABLE verifications.key_verifications_per_day_mv_v1
MODIFY QUERY
SELECT
	workspace_id,
	key_space_id,
	identity_id,
	key_id,
	outcome,
	count(*) AS COUNT,
	toStartOfDay(fromUnixTimestamp64Milli (time)) AS time,
	tags
FROM
	verifications.raw_key_verifications_v1
GROUP BY
	workspace_id,
	key_space_id,
	identity_id,
	key_id,
	outcome,
	time,
	tags
	;


ALTER TABLE verifications.key_verifications_per_month_mv_v1
MODIFY QUERY
SELECT
	workspace_id,
	key_space_id,
	identity_id,
	key_id,
	outcome,
	count(*) AS COUNT,
	toStartOfMonth(fromUnixTimestamp64Milli (time)) AS time,
	tags
FROM
	verifications.raw_key_verifications_v1
GROUP BY
	workspace_id,
	key_space_id,
	identity_id,
	key_id,
	outcome,
	time,
	tags
	;



-- +goose down



ALTER TABLE verifications.key_verifications_per_hour_mv_v1
MODIFY QUERY
  SELECT
    workspace_id,
    key_space_id,
    identity_id,
    key_id,
    outcome,
    count(*) as count,
    toStartOfHour(fromUnixTimestamp64Milli(time)) AS time
  FROM verifications.raw_key_verifications_v1
  GROUP BY
    workspace_id,
    key_space_id,
    identity_id,
    key_id,
    outcome,
    time
  ;


ALTER TABLE verifications.key_verifications_per_day_mv_v1
MODIFY QUERY
  SELECT
    workspace_id,
    key_space_id,
    identity_id,
    key_id,
    outcome,
    count(*) as count,
    toStartOfDay(fromUnixTimestamp64Milli(time)) AS time
    FROM verifications.raw_key_verifications_v1
  GROUP BY
    workspace_id,
    key_space_id,
    identity_id,
    key_id,
    outcome,
    time
  ;


ALTER TABLE verifications.key_verifications_per_month_mv_v1
MODIFY QUERY
  SELECT
    workspace_id,
    key_space_id,
    identity_id,
    key_id,
    outcome,
    count(*) as count,
    toStartOfMonth(fromUnixTimestamp64Milli(time)) AS time
  FROM verifications.raw_key_verifications_v1
  GROUP BY
    workspace_id,
    key_space_id,
    identity_id,
    key_id,
    outcome,
    time
  ;

ALTER TABLE verifications.key_verifications_per_hour_v1
DROP COLUMN tags;

ALTER TABLE verifications.key_verifications_per_day_v1
DROP COLUMN tags;

ALTER TABLE verifications.key_verifications_per_month_v1
DROP COLUMN tags;


ALTER TABLE verifications.raw_key_verifications_v1
DROP COLUMN tags;

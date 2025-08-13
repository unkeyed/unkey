-- +goose up
ALTER TABLE verifications.raw_key_verifications_v1
ADD COLUMN IF NOT EXISTS spent_credits Int64 DEFAULT 0;

-- +goose down
ALTER TABLE verifications.raw_key_verifications_v1
DROP COLUMN IF EXISTS spent_credits;

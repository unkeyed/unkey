-- +goose up
ALTER TABLE verifications.raw_key_verifications_v1
ADD COLUMN IF NOT EXISTS tags Array(String) DEFAULT [];


-- +goose down



ALTER TABLE verifications.raw_key_verifications_v1
DROP COLUMN IF EXISTS tags;

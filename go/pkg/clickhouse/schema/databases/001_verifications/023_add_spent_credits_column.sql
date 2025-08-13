ALTER TABLE verifications.raw_key_verifications_v1
ADD COLUMN IF NOT EXISTS spent_credits Int64 DEFAULT 0;

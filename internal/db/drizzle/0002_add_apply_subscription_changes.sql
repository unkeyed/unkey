-- Add apply_subscription_changes column to quota table
ALTER TABLE `quota` ADD COLUMN `apply_subscription_changes` BOOLEAN NOT NULL DEFAULT TRUE;
-- name: DeleteGithubAppInstallationsByInstallationId :exec
-- Removes every workspace binding when GitHub deletes an installation.
-- Safe to repeat if webhook processing retries after the deletion commits.
DELETE FROM github_app_installations WHERE installation_id = sqlc.arg(installation_id);

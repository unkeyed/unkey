-- name: SlackIDs :many
-- SlackIDs resolves GitHub logins to Slack user ids. Unmapped logins are simply
-- absent from the result.
SELECT github_login, slack_user_id
FROM user_map
WHERE github_login = ANY(sqlc.arg(logins)::text[]);

# Recover paused notifications

Notification handlers pause after 20 infrastructure failures. Resume the same
Restate invocation after you fix the dependency. Don't reset delivery rows or
submit a replacement event: either action can bypass duplicate protection.

This procedure uses the Restate 1.7.8 admin API. It needs operator access to the
affected cluster, not a customer API key. `Notify` can pause before a notification
row exists, so use Restate, not a scan of MySQL delivery statuses, to find work.

> Resume can send customer notifications. Confirm the cluster and invocation
> before the PATCH request. Never use restart-as-new, change the event ID, or
> change a delivery from `sending` or `unknown` to `pending` for recovery.

## Find paused work

Set `RESTATE_ADMIN_URL` to the approved cluster's admin endpoint. If that endpoint
requires authentication, supply `RESTATE_ADMIN_TOKEN` through your secret manager.
Don't enable shell tracing. These commands don't need a token for local tests.

```bash
set -euo pipefail
: "${RESTATE_ADMIN_URL:?Set the approved Restate admin endpoint}"
auth=(-H 'Accept: application/json')
if [[ -n "${RESTATE_ADMIN_TOKEN:-}" ]]; then
  auth+=(-H "Authorization: Bearer ${RESTATE_ADMIN_TOKEN}")
fi

mise exec -- jq -Rs '{query: .}' <<'SQL' |
SELECT id, status, target_service_name, target_service_key,
       target_handler_name, pinned_deployment_id, modified_at, last_failure
FROM sys_invocation
WHERE status = 'paused'
  AND ((target_service_name = 'notification.v1.NotificationService'
        AND target_handler_name = 'Notify')
    OR (target_service_name = 'notification.v1.DeliveryService'
        AND target_handler_name = 'Deliver'))
ORDER BY modified_at
SQL
curl --fail-with-body --silent --show-error "${auth[@]}" \
  -H 'Content-Type: application/json' \
  "${RESTATE_ADMIN_URL}/query" --data-binary @-
```

The response has a `rows` array. A `Notify` key is the notification ID. A `Deliver`
key is the delivery ID. Use the key to find the corresponding MySQL record by ID
when it exists. Don't print payloads or encrypted target snapshots while diagnosing.

Any paused row needs operator attention. Use this query for an alert before
enabling notifications in production; this runbook doesn't install that alert.
`pending` in MySQL alone isn't evidence of a pause: a delivery can be queued or
waiting for its retry time.

## Fix the cause

Inspect `last_failure` and repair MySQL, Vault, WorkOS, or worker availability
before resuming. Deterministic directory limits complete with a terminal error
instead of pausing. Find terminal planning failures with this separate query:

```bash
mise exec -- jq -Rs '{query: .}' <<'SQL' |
SELECT id, target_service_key, completion_failure, completed_at
FROM sys_invocation
WHERE status = 'completed' AND completion_result = 'failure'
  AND target_service_name = 'notification.v1.NotificationService'
  AND target_handler_name = 'Notify'
ORDER BY completed_at DESC
SQL
curl --fail-with-body --silent --show-error "${auth[@]}" \
  -H 'Content-Type: application/json' \
  "${RESTATE_ADMIN_URL}/query" --data-binary @-
```

Alert on new failed invocation IDs as well as paused work. Failed `Notify`
invocations might have no MySQL row, and Restate removes their metadata after
completion retention expires. Investigate `completion_failure` before that
retention ends. Completed failures cannot be resumed with this procedure.

Planning is atomic across channels. An invalid email audience can prevent webhook
targets for the same event from being planned. Keep audiences within the 1,000
member limit, or use explicit recipients. Explicit recipients don't require a
full organization scan. Live membership and opt-out checks still run before send.

## Resume one invocation

Copy one invocation ID from the result. Recheck its status and target immediately
before resuming. Keep the pinned worker deployment so the existing journal uses
the same handler code.

```bash
INVOCATION_ID='inv_REPLACE_ME'
[[ "$INVOCATION_ID" =~ ^inv_[A-Za-z0-9_-]+$ ]]

query="SELECT id, status, target_service_name, target_handler_name,
  pinned_deployment_id FROM sys_invocation WHERE id = '$INVOCATION_ID'"
before=$(mise exec -- jq -nc --arg query "$query" '{query: $query}' |
  curl --fail-with-body --silent --show-error "${auth[@]}" \
    -H 'Content-Type: application/json' \
    "${RESTATE_ADMIN_URL}/query" --data-binary @-)

mise exec -- jq -e '
  .rows | length == 1 and (.[0] |
    .status == "paused" and .pinned_deployment_id != null and
    ((.target_service_name == "notification.v1.NotificationService" and
      .target_handler_name == "Notify") or
     (.target_service_name == "notification.v1.DeliveryService" and
      .target_handler_name == "Deliver")))
' <<<"$before"

curl --fail-with-body --silent --show-error "${auth[@]}" \
  -X PATCH \
  "${RESTATE_ADMIN_URL}/invocations/${INVOCATION_ID}/resume?deployment=keep"
```

A successful resume returns HTTP 200 with an empty body. HTTP 409 means the
invocation can't be resumed, including when it has completed. If the request
times out, query its status before trying again. Don't change deployments or
create a new invocation to bypass an error.

## Verify the result

Repeat the query stored in `query` above. Confirm that the same invocation leaves
`paused` and retains its `pinned_deployment_id`. Check its delivery row as well:

| Delivery status | Meaning and action |
| --- | --- |
| `accepted` | The provider accepted the request. No further send is needed. |
| `suppressed` | An expiry, preference, destination, or membership check stopped delivery. |
| `failed` | The provider rejected the request or the retry budget ended. Investigate the reason. |
| `unknown` | An earlier attempt might have been accepted. Investigate the provider receipt; don't resend automatically. |

A pause before sending can recover to `accepted`. If a provider accepted the
request but the database write failed, Slack, Discord, and generic webhooks can
recover to `unknown` without another send. Resend can retry the same delivery ID
only within its idempotency window. Resume doesn't extend that window or an
event's expiry.

If the invocation pauses again, fix the reported cause rather than looping resume
requests. This procedure preserves the original journal, delivery ID, attempt
count, frozen content, and recipient checks. It doesn't guarantee delivery when
the provider's outcome is unknown.

import { displayNameOf } from "@/lib/auth/display-name";
import type { User } from "@/lib/auth/types";
import type { RollbackEvent } from "@/lib/collections/deploy/rollback";
import type { LatestAuditLogByTargetRow } from "@unkey/clickhouse/src/audit-logs";

// svc/ctrl rollback_handler.go writes the rolled-back-to deployment first and the
// replaced one second. A first id that is not live means a later deploy superseded it.
export function toRollbackEvent(
  row: LatestAuditLogByTargetRow,
  user: User | null,
  currentDeploymentId: string,
): RollbackEvent | null {
  const deploymentIds = row.targets.filter(([type]) => type === "deployment").map(([, id]) => id);
  if (deploymentIds[0] !== currentDeploymentId) {
    return null;
  }
  return {
    at: row.time,
    actor: actorOf(row, user),
    fromDeploymentId: deploymentIds[1] ?? null,
  };
}

function actorOf(row: LatestAuditLogByTargetRow, user: User | null): RollbackEvent["actor"] {
  if (row.actorType === "user" && user) {
    return { name: displayNameOf(user) ?? user.email, avatarUrl: user.avatarUrl };
  }
  return row.actorName ? { name: row.actorName, avatarUrl: null } : null;
}

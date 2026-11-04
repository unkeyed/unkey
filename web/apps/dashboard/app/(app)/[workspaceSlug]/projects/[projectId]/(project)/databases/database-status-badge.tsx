import { Badge } from "@unkey/ui";
import { type DatabaseStatus, statusCopy } from "./database-model";

export function DatabaseStatusBadge({ status }: { status: DatabaseStatus }) {
  const { badge, label } = statusCopy[status];
  if (!badge) {
    return null;
  }
  return (
    <Badge variant={badge} size="sm">
      {label}
    </Badge>
  );
}

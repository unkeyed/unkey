"use client";

import { match } from "@unkey/match";
import { Badge } from "@unkey/ui";
import type { DrainListItem } from "./drain-schema";

export function DrainStatusBadge({ status }: { status: DrainListItem["status"] }) {
  return match(status)
    .with("running", () => <Badge variant="success">Running</Badge>)
    .with("paused_by_user", () => <Badge variant="secondary">Paused</Badge>)
    .with("paused_by_failure", () => <Badge variant="error">Failing</Badge>)
    .exhaustive();
}

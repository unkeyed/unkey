import {
  type DeploymentStatus,
  type DeploymentStatusGroup,
  expandDeploymentStatusGroups,
  isDeploymentStatusGroup,
} from "@/lib/collections/deploy/deployment-status";
import type { Environment } from "@/lib/collections/deploy/environments";
import { parseDuration } from "@/lib/duration";
import type { DeploymentListFilterValue } from "../filters.schema";

export type DeploymentListInput = {
  statuses?: DeploymentStatus[];
  environmentIds?: string[];
  branches?: string[];
  startTime?: number;
  endTime?: number;
};

export type DeploymentListFilterInput = {
  input: DeploymentListInput;
  // A filter names an environment slug this app does not have or a status
  // that does not exist, so nothing can match. The caller renders the empty
  // state instead of querying.
  cannotMatch: boolean;
};

// Status values the previous filter bar wrote into URLs, mapped onto the
// groups that replaced them so old bookmarks keep working.
const LEGACY_STATUS_GROUPS: Record<string, DeploymentStatusGroup> = {
  pending: "queued",
  deploying: "building",
  skipped: "cancelled",
};

const MINUTE_MS = 60_000;

function stringValues(
  filters: DeploymentListFilterValue[],
  field: DeploymentListFilterValue["field"],
) {
  return filters.flatMap((f) =>
    f.field === field && typeof f.value === "string" ? [f.value] : [],
  );
}

function numberValue(
  filters: DeploymentListFilterValue[],
  field: DeploymentListFilterValue["field"],
) {
  const value = filters.find((f) => f.field === field)?.value;
  return typeof value === "number" ? value : undefined;
}

export function buildDeploymentListInput(
  filters: DeploymentListFilterValue[],
  environments: Environment[],
  now: number = Date.now(),
): DeploymentListFilterInput {
  const statusValues = stringValues(filters, "status").map(
    (value) => LEGACY_STATUS_GROUPS[value] ?? value,
  );
  const groups = statusValues.filter(isDeploymentStatusGroup);
  const statuses = expandDeploymentStatusGroups(groups);

  const slugs = stringValues(filters, "environment");
  const environmentIds = environments.filter((e) => slugs.includes(e.slug)).map((e) => e.id);

  const branches = stringValues(filters, "branch");

  const since = stringValues(filters, "since")[0];
  // Floored to the minute so the query input, and with it the cache key, stays
  // stable across renders within the same minute.
  const sinceStart =
    since !== undefined
      ? Math.floor((now - parseDuration(since)) / MINUTE_MS) * MINUTE_MS
      : undefined;
  const explicitStart = numberValue(filters, "startTime");
  const endTime = numberValue(filters, "endTime");
  const startTime =
    sinceStart !== undefined && explicitStart !== undefined
      ? Math.max(sinceStart, explicitStart)
      : (sinceStart ?? explicitStart);

  return {
    input: {
      ...(statuses.length > 0 && { statuses }),
      ...(environmentIds.length > 0 && { environmentIds }),
      ...(branches.length > 0 && { branches }),
      ...(startTime !== undefined && { startTime }),
      ...(endTime !== undefined && { endTime }),
    },
    cannotMatch:
      (slugs.length > 0 && environmentIds.length === 0) ||
      (statusValues.length > 0 && groups.length === 0),
  };
}

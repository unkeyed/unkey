import {
  DEPLOYMENT_STATUSES,
  type DeploymentStatus,
  type DeploymentStatusGroup,
  expandDeploymentStatusGroups,
  isDeploymentStatusGroup,
} from "@/lib/collections/deploy/deployment-status";
import type { Environment } from "@/lib/collections/deploy/environments";
import { getTimestampFromRelative } from "@/lib/duration";
import type { DeploymentListFilterValue } from "../filters.schema";

// startTime is inclusive and endTime exclusive, matching the API
export type DeploymentListFilter = {
  statuses: DeploymentStatus[];
  environmentId?: string;
  branches: string[];
  startTime?: number;
  endTime?: number;
};

export type DeploymentListFilterInput = {
  filter: DeploymentListFilter;
  // A filter names an environment slug this app does not have, a status that
  // does not exist, or an empty time range, so nothing can match. The caller
  // renders the empty state instead of querying.
  cannotMatch: boolean;
};

// Without picked statuses the list shows everything but skipped pushes.
const DEFAULT_STATUSES = DEPLOYMENT_STATUSES.filter((status) => status !== "skipped");

// Status values the previous filter bar wrote into URLs, mapped onto the
// groups that replaced them so old bookmarks keep working.
const LEGACY_STATUS_GROUPS: Record<string, DeploymentStatusGroup> = {
  pending: "queued",
  deploying: "building",
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
  const groups = [...new Set(statusValues.filter(isDeploymentStatusGroup))];
  const statuses = expandDeploymentStatusGroups(groups);

  const slugs = stringValues(filters, "environment");
  const matched = environments.filter((e) => slugs.includes(e.slug));

  const branches = stringValues(filters, "branch");

  const since = stringValues(filters, "since")[0];
  // Floored to the minute so the query input, and with it the cache key, stays
  // stable across renders within the same minute.
  const sinceStart =
    since !== undefined
      ? Math.floor(getTimestampFromRelative(since, now) / MINUTE_MS) * MINUTE_MS
      : undefined;
  const explicitStart = numberValue(filters, "startTime");
  const endTime = numberValue(filters, "endTime");
  const startTime =
    sinceStart !== undefined && explicitStart !== undefined
      ? Math.max(sinceStart, explicitStart)
      : (sinceStart ?? explicitStart);

  // Each app has one environment per kind, so picking every environment is the
  // same as picking none
  const environmentId = matched.length === 1 ? matched[0]?.id : undefined;

  return {
    filter: {
      statuses: statuses.length > 0 ? statuses : DEFAULT_STATUSES,
      ...(environmentId !== undefined && { environmentId }),
      branches,
      ...(startTime !== undefined && { startTime }),
      ...(endTime !== undefined && { endTime }),
    },
    cannotMatch:
      (slugs.length > 0 && matched.length === 0) ||
      (statusValues.length > 0 && groups.length === 0) ||
      (startTime !== undefined && endTime !== undefined && startTime >= endTime),
  };
}

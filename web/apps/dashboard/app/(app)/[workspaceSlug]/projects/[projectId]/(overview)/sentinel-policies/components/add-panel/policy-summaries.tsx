"use client";

import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { match } from "@unkey/match";
import type { ReactNode } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import type { MatchConditionFormValues, PolicyFormValues, RateLimitKeySource } from "./schema";

type KeyauthValues = Extract<PolicyFormValues, { type: "keyauth" }>;

const KEY_SOURCE_LABELS: Record<RateLimitKeySource, string> = {
  remoteIp: "IP",
  header: "Header",
  authenticatedSubject: "Subject",
  path: "Path",
  principalClaim: "Claim",
};

const Strong = ({ children, className }: { children: ReactNode; className?: string }) => (
  <span className={cn("text-gray-12 font-medium", className)}>{children}</span>
);

const Sep = () => <span className="text-gray-9 mx-1.5">·</span>;

export function summarizeMatchConditions(conditions: MatchConditionFormValues[]): ReactNode {
  if (conditions.length === 0) {
    return null;
  }
  return (
    <span className="text-gray-11">
      <Strong>{conditions.length}</Strong> condition{conditions.length === 1 ? "" : "s"}
    </span>
  );
}

function summarizeLocation(loc: KeyauthValues["locations"][number]): ReactNode {
  return match(loc.locationType)
    .with("bearer", () => <Strong>Bearer</Strong>)
    .with("header", () => (
      <>
        Header: <Strong>{loc.name || "—"}</Strong>
      </>
    ))
    .with("queryParam", () => (
      <>
        Query: <Strong>{loc.name || "—"}</Strong>
      </>
    ))
    .exhaustive();
}

export function summarizePolicy(
  values: PolicyFormValues,
  keyspaceNames?: Record<string, string>,
): ReactNode {
  return match(values)
    .with({ type: "keyauth" }, (v) => (
      <span className="text-gray-11">
        {v.keySpaceIds.length > 0 && v.keySpaceIds.length > 3 ? (
          <>
            <Strong>{v.keySpaceIds.length}</Strong> keyspaces
          </>
        ) : (
          <Strong className="inline-block max-w-50 truncate align-bottom">
            {v.keySpaceIds.map((id) => keyspaceNames?.[id] ?? id).join(", ")}
          </Strong>
        )}
        {v.locations.length === 1 && (
          <>
            <Sep />
            {summarizeLocation(v.locations[0])}
          </>
        )}
        {v.locations.length > 1 && (
          <>
            <Sep />
            <Strong>{v.locations.length}</Strong> key locations
          </>
        )}
      </span>
    ))
    .with({ type: "ratelimit" }, (v) => (
      <span className="text-gray-11">
        <Strong>{v.limit}</Strong> /{" "}
        {v.windowMs >= 1000 ? `${v.windowMs / 1000}s` : `${v.windowMs}ms`}
        <Sep />
        per <Strong>{KEY_SOURCE_LABELS[v.keySource]}</Strong>
        {v.keyValue && (
          <>
            : <Strong>{v.keyValue}</Strong>
          </>
        )}
      </span>
    ))
    .exhaustive();
}

/**
 * Live-subscribing wrapper around `summarizePolicy`. Reads only the fields the
 * summary actually renders so a keystroke in (say) the policy name field
 * re-renders just this small subtree, not the entire panel.
 */
function KeyauthPolicySummary() {
  const { control } = useFormContext<Extract<PolicyFormValues, { type: "keyauth" }>>();
  const keySpaceIds = useWatch({ control, name: "keySpaceIds" });
  const locations = useWatch({ control, name: "locations" });

  const { data: availableKeyspaces = {} } =
    trpc.deploy.environmentSettings.getAvailableKeyspaces.useQuery();
  const keyspaceNames: Record<string, string> = Object.fromEntries(
    Object.entries(availableKeyspaces).map(([id, ks]) => [id, ks?.api?.name ?? id]),
  );

  return (
    <div className="max-w-75 truncate">
      {summarizePolicy(
        {
          type: "keyauth",
          name: "",
          environmentId: "",
          matchConditions: [],
          keySpaceIds,
          locations,
          permissionQuery: "",
        },
        keyspaceNames,
      )}
    </div>
  );
}

function RatelimitPolicySummary() {
  const { control } = useFormContext<Extract<PolicyFormValues, { type: "ratelimit" }>>();
  const limit = useWatch({ control, name: "limit" });
  const windowMs = useWatch({ control, name: "windowMs" });
  const keySource = useWatch({ control, name: "keySource" });
  const keyValue = useWatch({ control, name: "keyValue" });

  return (
    <div className="max-w-75 truncate">
      {summarizePolicy({
        type: "ratelimit",
        name: "",
        environmentId: "",
        matchConditions: [],
        limit,
        windowMs,
        keySource,
        keyValue,
      })}
    </div>
  );
}

export function PolicySummary() {
  const { control } = useFormContext<PolicyFormValues>();
  const type = useWatch({ control, name: "type" });

  if (type === "ratelimit") {
    return <RatelimitPolicySummary />;
  }
  return <KeyauthPolicySummary />;
}

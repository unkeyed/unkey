"use client";

import { cn } from "@/lib/utils";
import { match } from "@unkey/match";
import type { ReactNode } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import type { MatchConditionFormValues, PolicyFormValues } from "./schema";

type KeyauthValues = Extract<PolicyFormValues, { type: "keyauth" }>;

const Strong = ({ children, className }: { children: ReactNode; className?: string }) => (
  <span className={cn("text-gray-12 font-medium", className)}>{children}</span>
);

const Sep = () => <span className="text-gray-9 mx-1.5">·</span>;

export function summarizeMatchConditions(conditions: MatchConditionFormValues[]): ReactNode {
  if (conditions.length === 0) {
    return <span className="text-gray-11">No conditions</span>;
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
        Key Auth
        {v.keySpaceIds.length > 0 && (
          <>
            <Sep />
            {v.keySpaceIds.length > 3 ? (
              <>
                <Strong>{v.keySpaceIds.length}</Strong> keyspaces
              </>
            ) : (
              <Strong className="inline-block max-w-[200px] truncate align-bottom">
                {v.keySpaceIds.map((id) => keyspaceNames?.[id] ?? id).join(", ")}
              </Strong>
            )}
          </>
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
    .exhaustive();
}

/**
 * Live-subscribing wrapper around `summarizePolicy`. Reads only the fields the
 * summary actually renders so a keystroke in (say) the policy name field
 * re-renders just this small subtree, not the entire panel.
 */
export function PolicySummary({ keyspaceNames }: { keyspaceNames: Record<string, string> }) {
  const { control } = useFormContext<PolicyFormValues>();
  const type = useWatch({ control, name: "type" });
  const name = useWatch({ control, name: "name" });
  const environmentId = useWatch({ control, name: "environmentId" });
  const keySpaceIds = useWatch({ control, name: "keySpaceIds" });
  const locations = useWatch({ control, name: "locations" });
  const permissionQuery = useWatch({ control, name: "permissionQuery" });

  return (
    <>
      {summarizePolicy(
        { type, name, environmentId, matchConditions: [], keySpaceIds, locations, permissionQuery },
        keyspaceNames,
      )}
    </>
  );
}

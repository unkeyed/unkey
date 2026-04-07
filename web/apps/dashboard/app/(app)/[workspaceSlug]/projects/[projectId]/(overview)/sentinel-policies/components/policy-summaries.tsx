import { match } from "@unkey/match";
import type { ReactNode } from "react";
import type { MatchConditionFormValues } from "./schema";
import type { PolicyFormValues } from "./schema";

type KeyauthValues = Extract<PolicyFormValues, { type: "keyauth" }>;

const Strong = ({ children }: { children: ReactNode }) => (
  <span className="text-gray-12 font-medium">{children}</span>
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

export function summarizePolicy(values: PolicyFormValues): ReactNode {
  return match(values)
    .with({ type: "keyauth" }, (v) => (
      <span className="text-gray-11">
        Key Auth
        {v.keySpaceIds.length > 0 && (
          <>
            <Sep />
            <Strong>{v.keySpaceIds.length}</Strong> keyspace{v.keySpaceIds.length === 1 ? "" : "s"}
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
    .with({ type: "ratelimit" }, (v) => {
      const seconds = v.windowMs >= 1000 ? `${(v.windowMs / 1000).toFixed(0)}s` : `${v.windowMs}ms`;
      return (
        <span className="text-gray-11">
          Ratelimit
          <Sep />
          <Strong>{v.limit}</Strong> req / <Strong>{seconds}</Strong>
          <Sep />
          {v.keySource}
        </span>
      );
    })
    .exhaustive();
}

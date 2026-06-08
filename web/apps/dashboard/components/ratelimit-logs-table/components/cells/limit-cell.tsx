import { safeParseJson } from "@/app/(app)/[workspaceSlug]/logs/utils";
import type { EnrichedRatelimitLog } from "../../hooks/use-ratelimit-logs-query";
import { EnrichmentSkeleton } from "./enrichment-skeleton";

export const LimitCell = ({ log }: { log: EnrichedRatelimitLog }) => {
  if (!log.isEnriched) {
    return <EnrichmentSkeleton />;
  }
  const body = safeParseJson(log.response_body);
  const parsedLimit = body?.limit ?? body?.data?.limit;
  // Treat a missing or zero limit as empty, matching DurationCell/ResetCell — a
  // limit of 0 reads as malformed/absent data here, not a meaningful value.
  return <div className="font-mono">{parsedLimit || "<EMPTY>"}</div>;
};

import { safeParseJson } from "@/app/(app)/[workspaceSlug]/logs/utils";
import type { EnrichedRatelimitLog } from "../../hooks/use-ratelimit-logs-query";
import { EnrichmentSkeleton } from "./enrichment-skeleton";

export const LimitCell = ({ log }: { log: EnrichedRatelimitLog }) => {
  if (!log.isEnriched) {
    return <EnrichmentSkeleton />;
  }
  const body = safeParseJson(log.response_body);
  const parsedLimit = body?.limit ?? body?.data?.limit;
  const isEmpty = parsedLimit == null || parsedLimit === 0;
  return <div className="font-mono">{isEmpty ? "<EMPTY>" : parsedLimit}</div>;
};

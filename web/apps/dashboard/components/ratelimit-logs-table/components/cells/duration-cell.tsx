import { safeParseJson } from "@/app/(app)/[workspaceSlug]/logs/utils";
import type { EnrichedRatelimitLog } from "../../hooks/use-ratelimit-logs-query";
import { EnrichmentSkeleton } from "./enrichment-skeleton";

const msToSeconds = (ms: number) => {
  const seconds = Math.round(ms / 1000);
  return `${seconds}s`;
};

export const DurationCell = ({ log }: { log: EnrichedRatelimitLog }) => {
  if (!log.isEnriched) {
    return <EnrichmentSkeleton />;
  }
  const parsedDuration = safeParseJson(log.request_body)?.duration;
  return (
    <div className="font-mono">{parsedDuration ? msToSeconds(parsedDuration) : "<EMPTY>"}</div>
  );
};

import type { EnrichedRatelimitLog } from "../../hooks/use-ratelimit-logs-query";
import { EnrichmentSkeleton } from "./enrichment-skeleton";

export const RegionValueCell = ({ log }: { log: EnrichedRatelimitLog }) => {
  if (!log.isEnriched) {
    return <EnrichmentSkeleton />;
  }
  return <div className="font-mono">{log.region}</div>;
};

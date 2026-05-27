import { safeParseJson } from "@/app/(app)/[workspaceSlug]/logs/utils";
import { cn } from "@/lib/utils";
import { TimestampInfo } from "@unkey/ui";
import type { EnrichedRatelimitLog } from "../../hooks/use-ratelimit-logs-query";
import { EnrichmentSkeleton } from "./enrichment-skeleton";

type ResetCellProps = {
  log: EnrichedRatelimitLog;
  pointerEventsNone: boolean;
};

export const ResetCell = ({ log, pointerEventsNone }: ResetCellProps) => {
  if (!log.isEnriched) {
    return <EnrichmentSkeleton />;
  }

  const body = safeParseJson(log.response_body);
  const parsedReset = body?.reset ?? body?.data?.reset;

  if (!parsedReset) {
    return <>{"<Empty>"}</>;
  }

  return (
    <div className="font-mono">
      <TimestampInfo
        value={parsedReset}
        className={cn(
          "font-mono group-hover:underline decoration-dotted",
          pointerEventsNone && "pointer-events-none",
        )}
      />
    </div>
  );
};

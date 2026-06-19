import { cn } from "@/lib/utils";
import { Badge } from "@unkey/ui";
import type { EnrichedRatelimitLog } from "../../hooks/use-ratelimit-logs-query";
import { BLOCKED_STATUS, getStatusStyle } from "../../utils/get-row-class";

type RatelimitStatusCellProps = {
  log: EnrichedRatelimitLog;
  isSelected: boolean;
};

export const RatelimitStatusCell = ({ log, isSelected }: RatelimitStatusCellProps) => {
  const style = getStatusStyle(log.status);

  return (
    <Badge
      className={cn(
        "uppercase px-1.5 rounded-md font-mono min-w-17.5 inline-block text-center",
        isSelected ? style.badge.selected : style.badge.default,
        "border-transparent",
      )}
    >
      {log.status === BLOCKED_STATUS ? "Blocked" : "Passed"}
    </Badge>
  );
};

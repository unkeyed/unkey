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
        "uppercase px-[6px] rounded-md font-mono min-w-[70px] inline-block text-center",
        isSelected ? style.badge.selected : style.badge.default,
      )}
    >
      {log.status === BLOCKED_STATUS ? "Blocked" : "Passed"}
    </Badge>
  );
};

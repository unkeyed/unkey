import type { AuditLog } from "@/lib/trpc/routers/audit/schema";
import { cn } from "@/lib/utils";
import { Badge } from "@unkey/ui";
import { getAuditStatusStyle, getEventType } from "../../utils/get-row-class";

type AuditActionBadgeCellProps = {
  log: AuditLog;
  isSelected: boolean;
};

export const AuditActionBadgeCell = ({ log, isSelected }: AuditActionBadgeCellProps) => {
  const eventType = getEventType(log.auditLog.event);
  const style = getAuditStatusStyle(log);

  return (
    <div className="flex items-center gap-3 group/action">
      <Badge
        className={cn(
          "uppercase px-[6px] rounded-md font-mono whitespace-nowrap",
          isSelected ? style.badge.selected : style.badge.default,
        )}
      >
        {eventType}
      </Badge>
    </div>
  );
};

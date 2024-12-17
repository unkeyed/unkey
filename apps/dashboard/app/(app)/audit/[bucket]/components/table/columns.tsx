import { TimestampInfo } from "@/components/timestamp-info";
import { Data } from "./types";
import { getEventType } from "./utils";
import { Column } from "@/components/virtual-table";
import { FunctionSquare, KeySquare } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { cn } from "@unkey/ui/src/lib/utils";

export const columns: Column<Data>[] = [
  {
    key: "time",
    header: "Time",
    width: "130px",
    render: (log) => (
      <TimestampInfo
        value={log.auditLog.time}
        className="font-mono group-hover:underline decoration-dotted"
      />
    ),
  },
  {
    key: "actor",
    header: "Actor",
    width: "10%",
    render: (log) => (
      <div className="flex items-center">
        {log.auditLog.actor.type === "user" && log.user ? (
          <div className="flex items-center w-full gap-2 max-sm:m-0 max-sm:gap-1 max-sm:text-xs">
            <span className="text-xs whitespace-nowrap">{`${
              log.user.firstName ?? ""
            } ${log.user.lastName ?? ""}`}</span>
          </div>
        ) : log.auditLog.actor.type === "key" ? (
          <div className="flex items-center w-full gap-2 max-sm:m-0 max-sm:gap-1 max-sm:text-xs">
            <KeySquare className="w-4 h-4" />
            <span className="font-mono text-xs">{log.auditLog.actor.id}</span>
          </div>
        ) : (
          <div className="flex items-center w-full gap-2 max-sm:m-0 max-sm:gap-1 max-sm:text-xs">
            <FunctionSquare className="w-4 h-4" />
            <span className="font-mono text-xs">{log.auditLog.actor.id}</span>
          </div>
        )}
      </div>
    ),
  },
  {
    key: "action",
    header: "Action",
    width: "72px",
    render: (log) => {
      const eventType = getEventType(log.auditLog.event);
      const badgeClassName = cn("font-mono capitalize", {
        "bg-error-3 text-error-11 hover:bg-error-4": eventType === "delete",
        "bg-warning-3 text-warning-11 hover:bg-warning-4":
          eventType === "update",
        "bg-success-3 text-success-11 hover:bg-success-4":
          eventType === "create",
        "bg-accent-3 text-accent-11 hover:bg-accent-4": eventType === "other",
      });
      return <Badge className={badgeClassName}>{eventType}</Badge>;
    },
  },
  {
    key: "event",
    header: "Event",
    width: "20%",
    render: (log) => (
      <div className="flex items-center gap-2 text-current font-mono text-xs">
        <span>{log.auditLog.event}</span>
      </div>
    ),
  },
  {
    key: "event-description",
    header: "Description",
    width: "auto",
    render: (log) => (
      <div className="text-current font-mono px-2 text-xs">
        {log.auditLog.description}
      </div>
    ),
  },
];

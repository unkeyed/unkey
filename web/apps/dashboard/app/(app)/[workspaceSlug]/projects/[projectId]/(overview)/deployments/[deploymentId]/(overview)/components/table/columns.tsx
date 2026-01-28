import type { Column } from "@/components/virtual-table/types";
import type { SentinelLogsResponseSchema } from "@unkey/clickhouse/src/sentinel";
import { TriangleWarning2 } from "@unkey/icons";
import { Badge, TimestampInfo } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { LatencyBadge } from "./components/latency-badge";
import { WARNING_ICON_STYLES, getStatusStyle } from "./utils/get-row-class";

export const columns: Column<SentinelLogsResponseSchema>[] = [
  {
    key: "time",
    header: "Time",
    width: "180px",
    headerClassName: "pl-8",
    render: (log) => (
      <div className="flex items-center gap-3 px-2">
        <WarningIcon status={log.response_status} />
        <div className="flex-1 min-w-0">
          <TimestampInfo
            value={log.time}
            className="font-mono group-hover:underline decoration-dotted"
          />
        </div>
      </div>
    ),
  },
  {
    key: "response_status",
    header: "Status",
    width: "120px",
    render: (log) => {
      const style = getStatusStyle(log.response_status);
      return (
        <Badge
          className={cn(
            "uppercase px-[6px] rounded-md font-mono whitespace-nowrap",
            style.badge.default,
          )}
        >
          {log.response_status}
        </Badge>
      );
    },
  },
  {
    key: "region",
    header: "Region",
    width: "100px",
    render: (log) => (
      <div className="font-mono pr-4 truncate uppercase" title={log.region}>
        {log.region}
      </div>
    ),
  },
  {
    key: "method",
    header: "Method",
    width: "80px",
    render: (log) => (
      <Badge
        className={cn(
          "uppercase px-[6px] rounded-md font-mono whitespace-nowrap",
          getStatusStyle(log.response_status).badge.default,
        )}
      >
        {log.method}
      </Badge>
    ),
  },
  {
    key: "path",
    header: "Path",
    width: "250px",
    render: (log) => (
      <div className="font-mono pr-4 truncate" title={log.path}>
        {log.path}
      </div>
    ),
  },
  {
    key: "latency",
    header: "Latency",
    width: "150px",
    render: (log) => <LatencyBadge log={log} />,
  },
];

const WarningIcon = ({ status }: { status: number }) => (
  <TriangleWarning2
    iconSize="md-regular"
    className={cn(
      WARNING_ICON_STYLES.base,
      status < 300 && "invisible",
      status >= 400 && status < 500 && WARNING_ICON_STYLES.warning,
      status >= 500 && WARNING_ICON_STYLES.error,
    )}
  />
);

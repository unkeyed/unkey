import { useRelativeTime } from "../../../../hooks/use-relative-time";

export interface TimestampCellProps {
  timestamp: number | Date;
  format?: "relative" | "absolute" | "both";
}

/**
 * Timestamp cell with relative or absolute time display
 */
export function TimestampCell({ timestamp, format = "relative" }: TimestampCellProps) {
  const date = typeof timestamp === "number" ? new Date(timestamp) : timestamp;
  const relative = useRelativeTime(date);

  if (Number.isNaN(date.getTime())) {
    return <span className="text-xs text-accent-9">—</span>;
  }

  if (format === "relative") {
    return (
      <span className="text-xs text-accent-11" title={date.toLocaleString()}>
        {relative}
      </span>
    );
  }

  if (format === "absolute") {
    return <span className="text-xs text-accent-11">{date.toLocaleString()}</span>;
  }

  // Both
  return (
    <div className="flex flex-col">
      <span className="text-xs text-accent-12">{relative}</span>
      <span className="text-xs text-accent-9">{date.toLocaleString()}</span>
    </div>
  );
}

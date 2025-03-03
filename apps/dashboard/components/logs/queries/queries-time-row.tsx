import { formatTimestampTooltip } from "@/components/logs/chart/utils/format-timestamp";
import { Clock } from "@unkey/icons";
import { QueriesPill } from "./queries-pill";
type TimeRowProps = {
  startTime?: number | null;
  endTime?: number | null;
  since?: string | null;
};

export const TimeRow = ({ startTime, endTime, since }: TimeRowProps) => {
  const startTimeFormated = typeof startTime === "number" ? formatTimestampTooltip(startTime) : "";
  const endTimeFormated = typeof endTime === "number" ? formatTimestampTooltip(endTime) : "";
  const sinceFormated = since ? since : "";

  const operator = since ? "since" : startTime && endTime ? "between" : "starts from";
  return startTime || endTime || since ? (
    <div className="flex items-center justify-start w-full gap-2">
      <div className="flex-col font-mono font-normal text-xs text-gray-9 align-start w-[43px]">
        Time
      </div>
      <Clock size="md-thin" />
      <span className="font-mono text-xs font-normal text-gray-9">{operator}</span>
      {startTime && <QueriesPill value={startTimeFormated} className="ellipsis" />}
      {endTime && <QueriesPill value={endTimeFormated} className="ellipsis" />}
      {since && <QueriesPill value={sinceFormated} className="ellipsis" />}
    </div>
  ) : null;
};

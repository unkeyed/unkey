import type { LogsFilterUrlValue } from "@/app/(app)/logs/filters.schema";
import { ChartActivity2 } from "@unkey/icons";
import { QueriesOverflow } from "./queries-overflow-tooltip";
import { QueriesPill } from "./queries-pill";

type StatusRowProps = {
  status: LogsFilterUrlValue[] | null;
};
export const StatusRow = ({ status }: StatusRowProps) => {
  const shownStatus = status && status.length <= 4 ? status : status?.slice(0, 3);
  const overflowStatus = status && status.length > 4 ? status.slice(3) : null;
  return shownStatus ? (
    <div className="flex flex-row items-center justify-start w-full gap-2">
      <div className="flex-col font-mono font-normal text-xs text-gray-9 align-start w-[43px]">
        Status
      </div>
      <ChartActivity2 className="size-3.5 mb-[2px]" />
      <span className="font-mono text-xs font-normal text-gray-9">{shownStatus[0]?.operator}</span>
      {shownStatus.map((item) => {
        return <QueriesPill value={item.value} />;
      })}
      {overflowStatus && overflowStatus.length > 0 ? (
        <QueriesOverflow list={overflowStatus} />
      ) : null}
    </div>
  ) : null;
};

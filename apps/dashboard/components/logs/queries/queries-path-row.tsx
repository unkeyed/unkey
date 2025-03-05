import type { LogsFilterUrlValue } from "@/app/(app)/logs/filters.schema";
import { Link4 } from "@unkey/icons";
import { QueriesOverflow } from "./queries-overflow-tooltip";
import { QueriesPill } from "./queries-pill";

type PathRowProps = {
  paths: LogsFilterUrlValue[] | null;
};
export const PathRow = ({ paths }: PathRowProps) => {
  return paths ? (
    <div className="flex flex-row items-center justify-start w-full gap-2">
      <div className="flex-col font-mono font-normal text-xs text-gray-9 align-start w-[43px]">
        Path
      </div>
      <Link4 className="size-3 ml-[1px]" />
      <span className="font-mono text-xs font-normal text-gray-9">{paths[0]?.operator}</span>
      <QueriesPill value={paths[0]?.value} />
      <QueriesOverflow list={paths.slice(1)} />
    </div>
  ) : null;
};

import type { LogsFilterUrlValue } from "@/app/(app)/logs/filters.schema";
import { Conversion } from "@unkey/icons";
import { QueriesPill } from "./queries-pill";

type MethodRowProps = {
  methods: LogsFilterUrlValue[] | null;
};
export const MethodRow = ({ methods }: MethodRowProps) => {
  return (
    methods &&
    methods.length > 0 && (
      <div className="flex flex-row items-center justify-start w-full gap-2 ellipsis">
        <div className="flex-col font-mono font-normal text-xs text-gray-9 align-start w-[43px]">
          Method
        </div>
        <Conversion className="size-3.5 mb-[2px] ml-[-1px]" />
        <span className="font-mono text-xs font-normal text-gray-9">{methods[0]?.operator}</span>
        {methods?.map((item) => (
          <QueriesPill value={item.value} />
        ))}
      </div>
    )
  );
};

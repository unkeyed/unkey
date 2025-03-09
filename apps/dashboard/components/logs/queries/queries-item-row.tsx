import type { LogsFilterField, LogsFilterUrlValue } from "@/app/(app)/logs/filters.schema";
import type { ReactNode } from "react";
import { QueriesPill } from "./queries-pill";

type QueriesItemRowProps = {
  list: LogsFilterUrlValue[] | null;
  field: LogsFilterField | "time";
  icon: ReactNode;
};

export const QueriesItemRow = ({ list, field, icon }: QueriesItemRowProps) => {
  if (!list) {
    return null;
  }
  const operator = () => {
    if (!list) {
      return null;
    }
    if (field === "time" && list.length > 1 && list[0].operator === "startsWith") {
      return "between";
    }
    if (field === "time" && list[0].value && list[0].operator === "startsWith") {
      return "starts from";
    }
    if (field === "time" && list[0].operator === "is") {
      return "since";
    }
    return list[0]?.operator;
  };

  if (!list || list?.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-row items-center justify-start w-full gap-2">
      <div className="flex-col font-mono font-normal text-xs text-gray-9 align-start w-[50px]">
        {field.charAt(0).toUpperCase() + field.slice(1)}
      </div>
      <div className="flex w-[20px] justify-center shrink-0 grow-0">{icon}</div>
      <span className="font-mono text-xs font-normal text-gray-9">{operator()}</span>
      {list?.map((item) => (
        <QueriesPill key={item.value} value={item.value} />
      ))}
    </div>
  );
};

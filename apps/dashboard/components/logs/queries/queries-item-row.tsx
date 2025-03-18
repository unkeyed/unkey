import { type ReactNode, useEffect, useState } from "react";
import { QueriesOverflow } from "./queries-overflow-tooltip";
import { QueriesPill } from "./queries-pill";
import { FieldsToTruncate } from "./utils";
type QueriesItemRowProps = {
  list: { value: string; color: string }[];
  field: string;
  Icon: ReactNode;
  operator: string;
};

export const QueriesItemRow = ({ list, field, Icon, operator }: QueriesItemRowProps) => {
  if (!list || list.length === 0) {
    return null;
  }
  const [firstItem, setFirstItem] = useState(list[0]);
  const [overflowList, setOverflowList] = useState(list.slice(1));
  const shouldTruncate = FieldsToTruncate.includes(field.toString());
  if (!list || list.length === 0) {
    return null;
  }

  useEffect(() => {
    setFirstItem(list[0]);
    setOverflowList(list.slice(1));
  }, [list]);

  return (
    <div className="flex flex-row items-center justify-start w-full gap-2">
      <div className="flex-col font-mono text-xs font-normal text-gray-9 align-start ">
        {field.charAt(0).toUpperCase() + field.slice(1)}
      </div>
      <div className="flex w-[20px] justify-center shrink-0 grow-0">{Icon}</div>
      <span className="font-mono text-xs font-normal text-gray-9">{operator}</span>
      <QueriesPill value={firstItem.value} color={firstItem.color} />
      {!shouldTruncate &&
        overflowList.map((item) => (
          <QueriesPill key={item.value} value={item.value} color={item.color} />
        ))}
      {shouldTruncate && <QueriesOverflow list={overflowList} />}
    </div>
  );
};

import type { ReactNode } from "react";
import { QueriesOverflow } from "./queries-overflow-tooltip";
import { QueriesPill } from "./queries-pill";

type ListType = { value: string; operator: string }[];

type QueriesItemRowProps = {
  list: ListType;
  field: string;
  Icon: ReactNode;
  operator: string;
};
const FieldsToTruncate = [
  "paths",
  "methods",
  "events",
  "identifiers",
  "requestIds",
  "rootKeys",
  "users",
  "bucket",
  "host",
  "requestId",
];
export const QueriesItemRow = ({ list, field, Icon, operator }: QueriesItemRowProps) => {
  const shouldTruncate = FieldsToTruncate.includes(field.toString());
  const overflowList = list.slice(1);
  // const [ newList, setNewList ] = useState<ListType | null>(null);
  if (!list || list.length === 0) {
    return null;
  }

  // if (list.length > 2 && shouldTruncate) {
  //   const newOverflowList = list.slice(1);
  //   setOverflowList(newOverflowList);
  //   const newlist = list.slice(0, 1);
  //   setNewList(newlist);
  // } else {
  //   setNewList(list);
  // }

  return (
    <div className="flex flex-row items-center justify-start w-full gap-2">
      <div className="flex-col font-mono font-normal text-xs text-gray-9 align-start w-[50px]">
        {field.charAt(0).toUpperCase() + field.slice(1)}
      </div>
      <div className="flex w-[20px] justify-center shrink-0 grow-0">{Icon}</div>
      <span className="font-mono text-xs font-normal text-gray-9">{operator}</span>
      <QueriesPill value={list[0].value} />
      {!shouldTruncate &&
        list.slice(1).map((item) => <QueriesPill key={item.value} value={item.value} />)}
      {shouldTruncate && overflowList && <QueriesOverflow list={overflowList} />}
      {/* {newList?.map((item) => (
        <QueriesPill key={item.value} value={item.value} />
      ))} */}
      {/* {shouldTruncate && overflowList ? <QueriesOverflow list={overflowList} /> : null} */}
    </div>
  );
};

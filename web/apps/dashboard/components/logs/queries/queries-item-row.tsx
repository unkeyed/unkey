import { useQueries } from "./queries-context";
import { QueriesOverflow } from "./queries-overflow-tooltip";
import { QueriesPill } from "./queries-pill";

type QueriesItemRowProps = {
  list: { value: string; color: string | null }[];
  field: string;
  operator: string;
  icon: React.ReactNode;
};

export const QueriesItemRow = ({ list, field, operator, icon }: QueriesItemRowProps) => {
  const { shouldTruncateRow } = useQueries();
  if (!list || list.length === 0 || list[0] === null) {
    return null;
  }

  const shouldTruncate = shouldTruncateRow(field);

  return (
    <>
      <div className="flex flex-row items-center justify-start w-full gap-2">
        <div className="flex-col font-mono text-xs font-normal text-gray-9 align-start">
          {field.charAt(0).toUpperCase() + field.slice(1)}
        </div>
        <div className="flex w-[20px] justify-center shrink-0 grow-0">{icon}</div>
        <span className="font-mono text-xs font-normal text-gray-9">{operator}</span>
        <QueriesPill value={list[0].value} color={list[0].color} />
        {field !== "time" &&
          !shouldTruncate &&
          list.slice(1).map((item) => {
            if (!item) {
              return null;
            }
            return <QueriesPill key={item.value} value={item.value} color={item.color} />;
          })}
        {shouldTruncate && list.slice(1).length > 0 && <QueriesOverflow list={list.slice(1)} />}
      </div>
      {field === "time" && list[1] && (
        <div className="flex flex-col ml-[123px] -mt-1 w-fit">
          <QueriesPill value={list[1].value} color={list[1].color} />
        </div>
      )}
    </>
  );
};

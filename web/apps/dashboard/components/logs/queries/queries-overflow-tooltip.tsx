import { InfoTooltip } from "@unkey/ui";
import { QueriesPill } from "./queries-pill";

type QueriesOverflowProps = {
  list: { value: string; color: string | null }[] | null;
};

export const QueriesOverflow = ({ list }: QueriesOverflowProps) => {
  if (!list || list.length === 0) {
    return null;
  }

  return (
    <InfoTooltip
      position={{ side: "bottom" }}
      content={
        <ul className="flex flex-col gap-2 p-2">
          {list?.map((item) => {
            return (
              <li key={item.value}>
                <QueriesPill value={item.value} color={item.color} />
              </li>
            );
          })}
        </ul>
      }
    >
      <QueriesPill value={`+${list.length} more`} className="text-gray-10" />
    </InfoTooltip>
  );
};

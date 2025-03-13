import { Tooltip, TooltipContent, TooltipTrigger } from "@unkey/ui";
import { QueriesPill } from "./queries-pill";

type QueriesOverflowProps = {
  list?: { value: string; operator: string }[];
};

export const QueriesOverflow = ({ list }: QueriesOverflowProps) => {
  if (!list || list.length === 0) {
    return null;
  }
  return (
    <Tooltip>
      <TooltipTrigger>
        <QueriesPill value={`+${list.length} more`} className="text-gray-10" />
      </TooltipTrigger>
      <TooltipContent
        className="flex h-full bg-white text-gray-12 rounded-lg font-500 text-[12px] justify-center items-center leading-6 shadow-lg"
        side="bottom"
      >
        <ul className="flex flex-col gap-2 p-2">
          {list?.map((item) => (
            <QueriesPill value={item.value} />
          ))}
        </ul>
      </TooltipContent>
    </Tooltip>
  );
};

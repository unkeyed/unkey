import { cn } from "@/lib/utils";
import { Key2, Page2 } from "@unkey/icons";

export const AssignedItemsCell = ({
  items,
  totalCount,
  type,
  isSelected = false,
}: {
  items: string[];
  totalCount?: number;
  type: "keys" | "permissions";
  isSelected?: boolean;
}) => {
  const hasMore = totalCount && totalCount > items.length;
  const icon =
    type === "keys" ? <Key2 size="md-regular" /> : <Page2 className="size-3" size="md-regular" />;

  const itemClassName = cn(
    "font-mono rounded-md py-[2px] px-1.5 items-center w-fit flex gap-2 transition-all duration-100 border border-dashed text-grayA-12",
    isSelected ? "bg-grayA-4 border-grayA-7" : "bg-grayA-3 border-grayA-6 group-hover:bg-grayA-4",
  );

  const emptyClassName = cn(
    "rounded-md py-[2px] px-1.5 items-center w-fit flex gap-2 transition-all duration-100 border border-dashed bg-grayA-2 ",
    isSelected ? "border-grayA-7 text-grayA-9" : "border-grayA-6 text-grayA-8",
  );

  if (items.length === 0) {
    return (
      <div className="flex flex-col gap-1 py-1 max-w-[200px]">
        <div className={emptyClassName}>
          {icon}
          <span className="text-grayA-9 text-xs">None assigned</span>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-1 py-2 max-w-[200px]">
      {items.map((item) => (
        <div className={itemClassName} key={item}>
          {icon}
          <span className="text-grayA-11 text-xs max-w-[150px] truncate">{item}</span>
        </div>
      ))}
      {hasMore && (
        <div className={itemClassName}>
          <span className="text-grayA-9 text-xs max-w-[150px] truncate">
            {totalCount - items.length} more {type}...
          </span>
        </div>
      )}
    </div>
  );
};

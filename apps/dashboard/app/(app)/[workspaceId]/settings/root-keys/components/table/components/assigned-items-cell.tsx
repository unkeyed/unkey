import { cn } from "@/lib/utils";
import { Page2 } from "@unkey/icons";

export const AssignedItemsCell = ({
  permissionSummary,
  isSelected = false,
}: {
  permissionSummary: {
    total: number;
    categories: Record<string, number>;
  };
  isSelected?: boolean;
}) => {
  const { total } = permissionSummary;

  const itemClassName = cn(
    "font-mono rounded-md py-[2px] px-1.5 items-center w-fit flex gap-2 transition-all duration-100 border border-dashed text-grayA-12",
    isSelected ? "bg-grayA-4 border-grayA-7" : "bg-grayA-3 border-grayA-6 group-hover:bg-grayA-4",
  );

  const emptyClassName = cn(
    "rounded-md py-[2px] px-1.5 items-center w-fit flex gap-2 transition-all duration-100 border border-dashed bg-grayA-2",
    isSelected ? "border-grayA-7 text-grayA-9" : "border-grayA-6 text-grayA-8",
  );

  if (total === 0) {
    return (
      <div className="flex flex-col gap-1 py-1 max-w-[300px]">
        <div className={emptyClassName}>
          <Page2 className="size-3" />
          <span className="text-grayA-9 text-xs">None assigned</span>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-wrap gap-1 py-2 max-w-[300px]">
      <div className={itemClassName}>
        <Page2 className="size-3" />
        <span className="text-grayA-11 text-xs max-w-[150px] truncate">
          {total} Permission{total === 1 ? "" : "s"}
        </span>
      </div>
    </div>
  );
};

import { cn } from "@/lib/utils";
import { Page2 } from "@unkey/icons";
import { CopyButton } from "@unkey/ui";

export type SlugCellProps = {
  value?: string;
  isSelected?: boolean;
};

export const SlugCell = ({ value, isSelected = false }: SlugCellProps) => {
  if (!value) {
    return (
      <div className="flex flex-col gap-1 py-2 max-w-[200px]">
        <div
          className={cn(
            "rounded-md py-[2px] px-1.5 items-center w-fit flex gap-2 transition-all duration-100 border border-dashed bg-grayA-2",
            isSelected ? "border-grayA-7 text-grayA-9" : "border-grayA-6 text-grayA-8",
          )}
        >
          <Page2 iconSize="md-medium" className="opacity-50" />
          <span className="text-grayA-9 text-xs">No slug</span>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-1 py-2 max-w-[200px]">
      <div
        className={cn(
          "group font-mono rounded-md py-[2px] px-1.5 items-center w-fit flex gap-2 transition-all duration-100 border border-dashed text-grayA-12",
          isSelected
            ? "bg-grayA-4 border-grayA-7"
            : "bg-grayA-3 border-grayA-6 group-hover:bg-grayA-4",
        )}
      >
        <Page2 iconSize="md-medium" className="opacity-50" />
        <div className="text-grayA-11 text-xs max-w-[150px] truncate" title={value}>
          {value}
        </div>
        <CopyButton
          value={value}
          variant="ghost"
          className="h-4 w-4 opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 focus-visible:opacity-100 transition-opacity"
        />
      </div>
    </div>
  );
};

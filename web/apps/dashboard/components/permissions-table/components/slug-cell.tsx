import { IconPage2Outline18 } from "@unkey/icons";
import { CopyButton } from "@unkey/ui";
import { cn } from "cn";

export type SlugCellProps = {
  value?: string;
  isSelected?: boolean;
};

export const SlugCell = ({ value, isSelected = false }: SlugCellProps) => {
  if (!value) {
    return (
      <div className="flex flex-col gap-1 py-2">
        <div
          className={cn(
            "rounded-md py-[2px] px-1.5 items-center w-fit flex gap-2 transition-all duration-100 border border-dashed bg-grayA-2",
            isSelected ? "border-strong text-grayA-9" : "border-input text-grayA-8",
          )}
        >
          <IconPage2Outline18 className="size-3.5 opacity-50" />
          <span className="text-grayA-9 text-xs">No slug</span>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-1 py-2">
      <div
        className={cn(
          "group font-mono rounded-md py-[2px] px-1.5 items-center w-fit max-w-full flex gap-2 transition-all duration-100 border border-dashed text-grayA-12",
          isSelected
            ? "bg-grayA-4 border-strong"
            : "bg-grayA-3 border-input group-hover:bg-grayA-4",
        )}
      >
        <IconPage2Outline18 className="size-3.5 shrink-0 opacity-50" />
        <div className="text-grayA-11 text-xs min-w-0 truncate" title={value}>
          {value}
        </div>
        <CopyButton
          value={value}
          variant="ghost"
          className="h-4 w-4 shrink-0 opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 focus-visible:opacity-100 transition-opacity"
        />
      </div>
    </div>
  );
};

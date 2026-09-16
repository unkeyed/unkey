import { IconCircleLockOutline18 } from "@unkey/icons";
import { cn } from "../../../../lib/utils";
import { toast } from "../../../toaster";

export interface HiddenValueCellProps {
  start: string;
  end?: string;
  title: string;
  selected: boolean;
}

export const HiddenValueCell = ({
  start,
  end = "",
  title = "Value",
  selected,
}: HiddenValueCellProps) => {
  const displayValue = `${start}••••${end}`;

  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    navigator.clipboard
      .writeText(displayValue)
      .then(() => {
        toast.success(`${title} copied to clipboard`);
      })
      .catch((error) => {
        console.error("Failed to copy to clipboard:", error);
        toast.error("Failed to copy to clipboard");
      });
  };

  return (
    <>
      {/* biome-ignore lint/a11y/useKeyWithClickEvents: <explanation> */}
      <div
        className={cn(
          "rounded-lg border bg-white dark:bg-base-12 border-accent-4 text-grayA-11 w-[150px] px-2 py-1 flex gap-2 items-center cursor-pointer h-[28px] group-hover:border-grayA-3 font-mono",
          selected && "border-grayA-3",
        )}
        onClick={(e) => handleClick(e)}
      >
        <div>
          <IconCircleLockOutline18 className="size-3 text-gray-9" />
        </div>
        <div>{displayValue}</div>
      </div>
    </>
  );
};

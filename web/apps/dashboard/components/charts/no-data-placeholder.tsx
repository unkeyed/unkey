import { IconChartBarAxisYOutline18 } from "@unkey/icons";
import { cn } from "cn";

const ICON_MIN_HEIGHT = 64;

export function NoDataPlaceholder({
  height,
  className,
  inert,
}: {
  height: number;
  className?: string;
  inert?: boolean;
}) {
  return (
    <div
      inert={inert}
      className={cn(
        "flex w-full flex-col items-center justify-center gap-1.5 rounded-md border border-dashed border-grayA-5",
        className,
      )}
      style={{ height }}
    >
      {height >= ICON_MIN_HEIGHT && (
        <IconChartBarAxisYOutline18 className="size-4 text-gray-8" aria-hidden />
      )}
      <p className="text-xs text-gray-11">No data in this range</p>
    </div>
  );
}

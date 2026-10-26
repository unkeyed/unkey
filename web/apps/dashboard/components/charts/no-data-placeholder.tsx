import { IconChartBarAxisYOutline18 } from "@unkey/icons";

const ICON_MIN_HEIGHT = 64;

export function NoDataPlaceholder({ height }: { height: number }) {
  return (
    <div
      className="flex w-full flex-col items-center justify-center gap-1.5 rounded-md border border-dashed border-grayA-5"
      style={{ height }}
    >
      {height >= ICON_MIN_HEIGHT && (
        <IconChartBarAxisYOutline18 className="size-4 text-gray-8" aria-hidden />
      )}
      <p className="text-xs text-gray-11">No data in this range</p>
    </div>
  );
}

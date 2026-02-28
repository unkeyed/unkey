import { CircleCaretRight } from "@unkey/icons";

/**
 * Separator component for real-time data boundary
 * Preserves exact design from virtual-table
 */
export const RealtimeSeparator = () => {
  return (
    <div className="h-[26px] bg-info-2 font-mono text-xs text-info-11 rounded-md flex items-center gap-3 px-2">
      <CircleCaretRight className="size-3" />
      Live
    </div>
  );
};

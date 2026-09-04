import { Dots } from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";

export const ActionColumnSkeleton = () => (
  <button
    type="button"
    className={cn(
      "group size-5 p-0 rounded-sm m-0 items-center flex justify-center",
      "border border-gray-6",
    )}
    disabled
  >
    <Dots className="text-gray-11 opacity-50" iconSize="sm-regular" />
  </button>
);

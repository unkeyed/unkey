import { Dots } from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";

export const TableActionButton = () => {
  return (
    <button
      type="button"
      className={cn(
        "group-data-[state=open]:bg-gray-6 group-hover:bg-gray-6 group size-5 p-0 rounded m-0 items-center flex justify-center",
        "border border-gray-6 group-hover:border-gray-8 ring-2 ring-transparent focus-visible:ring-gray-7 focus-visible:border-gray-7",
      )}
    >
      <Dots className="group-hover:text-gray-12 text-gray-11" size="sm-regular" />
    </button>
  );
};

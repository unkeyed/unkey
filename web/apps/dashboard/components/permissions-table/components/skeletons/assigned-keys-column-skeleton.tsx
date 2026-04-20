import { Key2 } from "@unkey/icons";

export const AssignedKeysColumnSkeleton = () => (
  <div className="flex flex-col gap-1 py-2 max-w-[200px]">
    <div className="rounded-md py-[2px] px-1.5 items-center w-fit flex gap-2 border border-dashed bg-grayA-3 border-grayA-6 animate-pulse h-[22px]">
      <Key2 iconSize="md-medium" className="opacity-50" />
      <div className="h-2 w-16 bg-grayA-3 rounded-sm animate-pulse" />
    </div>
  </div>
);

import { Tag } from "@unkey/icons";

export const RoleColumnSkeleton = () => (
  <div className="flex flex-col items-start px-[18px] py-[6px]">
    <div className="flex gap-4 items-center">
      <div className="size-5 rounded-sm flex items-center justify-center border border-grayA-3 bg-grayA-3 animate-pulse">
        <Tag iconSize="sm-regular" className="text-gray-12 opacity-50" />
      </div>
      <div className="flex flex-col gap-1">
        <div className="h-4 w-24 bg-grayA-3 rounded-sm animate-pulse" />
        <div className="h-4 w-32 bg-grayA-3 rounded-sm animate-pulse" />
      </div>
    </div>
  </div>
);

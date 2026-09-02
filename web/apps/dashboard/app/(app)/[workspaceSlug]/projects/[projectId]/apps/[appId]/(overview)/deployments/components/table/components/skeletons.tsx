import { cn } from "@unkey/ui/src/lib/utils";
import { IconCloudOutline12, IconDotsOutline12 } from "nucleo-ui-outline-12";
import { IconCubeOutline18 } from "nucleo-ui-outline-18";

export const DeploymentIdColumnSkeleton = () => (
  <div className="flex flex-col items-start px-[18px] py-[12px]">
    <div className="flex gap-5 items-center w-full">
      <div className="size-5 rounded-sm flex items-center justify-center border border-grayA-3 bg-grayA-3 animate-pulse">
        <IconCloudOutline12 className="text-gray-12 opacity-50" />
      </div>
      <div className="w-[200px]">
        <div className="h-4 w-32 bg-grayA-3 rounded-sm animate-pulse mb-1" />
        <div className="h-3 w-24 bg-grayA-3 rounded-sm animate-pulse" />
      </div>
    </div>
  </div>
);

export const EnvColumnSkeleton = () => (
  <div className="bg-grayA-3 text-xs items-center flex gap-2 p-1.5 rounded-md relative w-fit">
    <div className="h-3 w-16 bg-grayA-4 rounded-sm" />
  </div>
);

export const StatusColumnSkeleton = () => (
  <div className="bg-grayA-3 items-center flex gap-2 p-1.5 rounded-md w-fit relative">
    <div className="size-4 bg-grayA-4 rounded-full" />
    <div className="h-3 w-12 bg-grayA-4 rounded-sm" />
  </div>
);

export const InstancesColumnSkeleton = () => (
  <div className="bg-grayA-3 font-mono text-xs items-center flex gap-2 p-1.5 rounded-md relative text-grayA-11 w-fit">
    <IconCubeOutline18 className="size-3 text-gray-12 opacity-50" />
    <div className="flex gap-0.5">
      <div className="h-3 w-4 bg-grayA-4 rounded-sm tabular-nums" />
      <div className="h-3 w-6 bg-grayA-4 rounded-sm" />
    </div>
  </div>
);

export const CreatedAtColumnSkeleton = () => (
  <div className="h-4 w-24 bg-grayA-3 rounded-sm font-mono" />
);

export const ActionColumnSkeleton = () => (
  <button
    type="button"
    className={cn(
      "group size-5 p-0 rounded-sm m-0 items-center flex justify-center",
      "border border-gray-6",
    )}
    disabled
  >
    <IconDotsOutline12 className="text-gray-11 opacity-50" />
  </button>
);

export const DomainListSkeleton = () => (
  <div className="flex items-center gap-2">
    <div className="h-4 w-36 bg-grayA-3 rounded-sm animate-pulse" />
    <div className="rounded-full px-1.5 py-0.5 bg-grayA-3 animate-pulse size-[22px] flex items-center justify-center">
      <div className="size-3 bg-grayA-3 rounded-full animate-pulse" />
    </div>
  </div>
);

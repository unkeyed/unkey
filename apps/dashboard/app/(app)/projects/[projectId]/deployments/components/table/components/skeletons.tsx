import { Cloud, CodeBranch, Cube, Dots } from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";

export const DeploymentIdColumnSkeleton = () => (
  <div className="flex flex-col items-start px-[18px] py-[12px]">
    <div className="flex gap-5 items-center w-full">
      <div className="size-5 rounded flex items-center justify-center border border-grayA-3 bg-grayA-3 animate-pulse">
        <Cloud iconsize="sm-regular" className="text-gray-12 opacity-50" />
      </div>
      <div className="w-[200px]">
        <div className="h-4 w-32 bg-grayA-3 rounded animate-pulse mb-1" />
        <div className="h-3 w-24 bg-grayA-3 rounded animate-pulse" />
      </div>
    </div>
  </div>
);

export const EnvColumnSkeleton = () => (
  <div className="bg-grayA-3 text-xs items-center flex gap-2 p-1.5 rounded-md relative w-fit">
    <div className="h-3 w-16 bg-grayA-4 rounded" />
  </div>
);

export const StatusColumnSkeleton = () => (
  <div className="bg-grayA-3 items-center flex gap-2 p-1.5 rounded-md w-fit relative">
    <div className="size-4 bg-grayA-4 rounded-full" />
    <div className="h-3 w-12 bg-grayA-4 rounded" />
  </div>
);

export const InstancesColumnSkeleton = () => (
  <div className="bg-grayA-3 font-mono text-xs items-center flex gap-2 p-1.5 rounded-md relative text-grayA-11 w-fit">
    <Cube className="text-gray-12 opacity-50" iconsize="sm-regular" />
    <div className="flex gap-0.5">
      <div className="h-3 w-4 bg-grayA-4 rounded tabular-nums" />
      <div className="h-3 w-6 bg-grayA-4 rounded" />
    </div>
  </div>
);

export const SizeColumnSkeleton = () => (
  <div className="bg-grayA-3 font-mono text-xs items-center flex gap-2 p-1.5 rounded-md relative text-grayA-11 w-fit">
    <Cube className="text-gray-12 opacity-50" iconsize="sm-regular" />
    <div className="flex gap-1">
      <div className="h-3 w-8 bg-grayA-4 rounded" />
      <div className="h-3 w-8 bg-grayA-4 rounded tabular-nums" />
    </div>
  </div>
);

export const SourceColumnSkeleton = () => (
  <div className="flex flex-col items-start px-[18px] py-1.5">
    <div className="flex gap-5 items-center w-full">
      <div className="size-5 rounded flex items-center justify-center border border-grayA-3 bg-grayA-3">
        <CodeBranch iconsize="sm-regular" className="text-gray-12 opacity-50" />
      </div>
      <div className="w-[200px]">
        <div className="flex items-center gap-2 mb-1">
          <div className="h-[13px] w-16 bg-grayA-4 rounded font-mono leading-5" />
        </div>
        <div className="h-3 w-24 bg-grayA-4 rounded font-mono mt-1" />
      </div>
    </div>
  </div>
);

export const CreatedAtColumnSkeleton = () => (
  <div className="h-4 w-24 bg-grayA-3 rounded font-mono" />
);

export const AuthorColumnSkeleton = () => (
  <div className="flex items-center gap-2">
    <div className="rounded-full size-5 bg-grayA-3" />
    <div className="h-3 w-20 bg-grayA-3 rounded font-medium text-xs" />
  </div>
);

export const ActionColumnSkeleton = () => (
  <button
    type="button"
    className={cn(
      "group size-5 p-0 rounded m-0 items-center flex justify-center",
      "border border-gray-6"
    )}
    disabled
  >
    <Dots className="text-gray-11 opacity-50" iconsize="sm-regular" />
  </button>
);

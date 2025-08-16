import { cn } from "@/lib/utils";
import { Cloud, CodeBranch, CodeCommit, Cube, Dots } from "@unkey/icons";

export const DeploymentIdColumnSkeleton = () => (
  <div className="flex flex-col items-start px-[18px] py-[12px]">
    <div className="flex gap-5 items-center w-full">
      <div className="size-5 rounded flex items-center justify-center border border-grayA-3 bg-grayA-3 animate-pulse">
        <Cloud size="sm-regular" className="text-gray-12 opacity-50" />
      </div>
      <div className="w-[200px]">
        <div className="h-4 w-32 bg-grayA-3 rounded animate-pulse mb-1" />
        <div className="h-3 w-24 bg-grayA-3 rounded animate-pulse" />
      </div>
    </div>
  </div>
);

export const EnvColumnSkeleton = () => (
  <div className="bg-grayA-3 font-mono text-xs items-center flex gap-2 p-1.5 rounded-md relative w-fit animate-pulse">
    <div className="h-3 w-16 bg-grayA-4 rounded" />
  </div>
);

export const StatusColumnSkeleton = () => (
  <div className="bg-grayA-3 items-center flex gap-2 p-1.5 rounded-md w-fit relative animate-pulse">
    <div className="size-4 bg-grayA-4 rounded-full" />
    <div className="h-3 w-12 bg-grayA-4 rounded" />
  </div>
);

export const InstancesColumnSkeleton = () => (
  <div className="bg-grayA-3 font-mono text-xs items-center flex gap-2 p-1.5 rounded-md relative w-fit animate-pulse">
    <Cube className="text-gray-12 opacity-50" size="sm-regular" />
    <div className="h-3 w-8 bg-grayA-4 rounded" />
  </div>
);

export const SourceColumnSkeleton = () => (
  <div className="flex items-center gap-1">
    <div className="bg-grayA-3 text-xs items-center flex gap-2 p-1.5 rounded-md relative w-fit animate-pulse">
      <CodeBranch className="text-gray-12 opacity-50" size="sm-regular" />
      <div className="h-3 w-12 bg-grayA-4 rounded" />
    </div>
    <div className="bg-grayA-3 text-xs items-center flex gap-2 p-1.5 rounded-md relative w-fit shrink-0 animate-pulse">
      <CodeCommit className="text-gray-12 opacity-50 rotate-90 shrink-0" size="md-bold" />
      <div className="h-3 w-16 bg-grayA-4 rounded" />
    </div>
  </div>
);

export const CreatedAtColumnSkeleton = () => (
  <div className="flex flex-col items-start py-[6px]">
    <div className="h-4 w-24 bg-grayA-3 rounded animate-pulse" />
  </div>
);

export const AuthorColumnSkeleton = () => (
  <div className="flex items-center gap-2 animate-pulse">
    <div className="rounded-full size-5 bg-grayA-3 flex items-center justify-center">
      <div className="rounded-full size-3 bg-grayA-4" />
    </div>
    <div className="h-3 w-20 bg-grayA-3 rounded" />
  </div>
);

export const ActionColumnSkeleton = () => (
  <button
    type="button"
    className={cn(
      "group size-5 p-0 rounded m-0 items-center flex justify-center animate-pulse",
      "border border-gray-6",
    )}
  >
    <Dots className="text-gray-11 opacity-50" size="sm-regular" />
  </button>
);

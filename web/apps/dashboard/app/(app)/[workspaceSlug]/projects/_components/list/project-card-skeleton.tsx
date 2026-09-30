import { Skeleton } from "@unkey/ui";

export function ProjectCardSkeleton() {
  return (
    <div className="flex h-full min-h-[124px] w-full flex-col gap-4 rounded-lg border bg-raised shadow-xs p-5">
      <div className="flex min-h-5 items-center gap-2.5">
        <Skeleton className="size-7 shrink-0 rounded-lg" />
        <Skeleton className="h-3.5 w-28" />
        <Skeleton className="ml-auto size-6 shrink-0 rounded-md" />
      </div>

      <div className="flex flex-col gap-1.5">
        <div className="flex h-6 items-center justify-between gap-2">
          <Skeleton className="h-3.5 w-20" />
          <Skeleton className="h-3.5 w-24" />
        </div>
        <div className="flex h-6 items-center justify-between gap-2">
          <Skeleton className="h-3.5 w-28" />
          <Skeleton className="h-3.5 w-20" />
        </div>
      </div>
    </div>
  );
}

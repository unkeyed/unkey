import { Skeleton } from "@unkey/ui";

// Mirrors ApiListCard's layout exactly (same paddings, header line heights, and the
// h-12 chart well) so swapping the list-loading skeleton for real cards causes no
// layout shift.
export function ApiCardSkeleton() {
  return (
    <div className="relative h-full p-5 flex flex-col border border-grayA-4 rounded-lg w-full gap-5">
      <div className="flex flex-col w-full gap-2 min-w-0">
        <Skeleton className="h-[14px] w-32" />
        <Skeleton className="h-[12px] w-40" />
      </div>
      <div className="mt-auto flex flex-col gap-3">
        <Skeleton className="h-12 w-full" />
        <div className="flex h-4 gap-3 items-center min-w-0 text-xs">
          <Skeleton className="h-3 w-16" />
        </div>
      </div>
    </div>
  );
}

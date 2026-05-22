import { Card, Skeleton } from "@unkey/ui";

export function SkeletonList() {
  return (
    <Card className="w-full max-w-2xl">
      <div className="divide-y divide-border">
        <div className="flex items-center gap-4 px-5 py-4">
          <Skeleton className="size-8 shrink-0 rounded-full" />
          <div className="flex min-w-0 flex-1 flex-col gap-2">
            <Skeleton className="h-4 w-2/3" />
            <Skeleton className="h-3 w-1/3" />
          </div>
          <Skeleton className="h-3 w-20 shrink-0" />
        </div>
        <div className="flex items-center gap-4 px-5 py-4">
          <Skeleton className="size-8 shrink-0 rounded-full" />
          <div className="flex min-w-0 flex-1 flex-col gap-2">
            <Skeleton className="h-4 w-3/4" />
            <Skeleton className="h-3 w-2/5" />
          </div>
          <Skeleton className="h-3 w-20 shrink-0" />
        </div>
        <div className="flex items-center gap-4 px-5 py-4">
          <Skeleton className="size-8 shrink-0 rounded-full" />
          <div className="flex min-w-0 flex-1 flex-col gap-2">
            <Skeleton className="h-4 w-1/2" />
            <Skeleton className="h-3 w-1/4" />
          </div>
          <Skeleton className="h-3 w-20 shrink-0" />
        </div>
      </div>
    </Card>
  );
}

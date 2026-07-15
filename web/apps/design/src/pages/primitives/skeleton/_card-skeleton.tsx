import { Card, CardContent, CardHeader, Skeleton } from "@unkey/ui";

export function SkeletonCard() {
  return (
    <Card className="w-full max-w-xs">
      <CardHeader className="flex-row items-center gap-4">
        <Skeleton className="size-10 rounded-[10px]" />
        <div className="flex min-w-0 flex-1 flex-col gap-2">
          <Skeleton className="h-3.5 w-24" />
          <Skeleton className="h-3 w-32" />
        </div>
      </CardHeader>
      <CardContent className="flex flex-col gap-2">
        <Skeleton className="h-5 w-40" />
        <div className="flex items-center justify-between">
          <Skeleton className="h-4 w-16" />
          <Skeleton className="h-4 w-16" />
        </div>
      </CardContent>
    </Card>
  );
}

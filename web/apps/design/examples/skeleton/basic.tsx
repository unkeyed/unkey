import { Skeleton } from "@unkey/ui";

export default function BasicSkeleton() {
  return (
    <div className="flex justify-center">
      <Skeleton className="h-4 w-32" />
    </div>
  );
}

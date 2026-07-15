import { Skeleton } from "@unkey/ui";

// biome-ignore lint/style/noDefaultExport: Blume's Component renderer imports examples by default export.
export default function BasicSkeletonExample() {
  return (
    <div className="flex w-full items-center justify-center p-8">
      <Skeleton className="h-4 w-32" />
    </div>
  );
}

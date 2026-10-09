import { ResourceListBody, ResourceListContent, ResourceListItem, Skeleton } from "@unkey/ui";

const ROWS = ["a", "b", "c", "d", "e", "f"];

export function EnvVarsSkeleton() {
  return (
    <ResourceListContent aria-busy="true">
      <ResourceListBody aria-hidden="true">
        {ROWS.map((key) => (
          <ResourceListItem key={key} className="flex h-12 items-center gap-4 px-4">
            <Skeleton className="size-6 shrink-0 rounded-md" />
            <Skeleton className="h-3 w-40" />
            <Skeleton className="h-3 w-24" />
            <Skeleton className="ml-auto h-3 w-20" />
            <Skeleton className="h-3 w-24" />
          </ResourceListItem>
        ))}
      </ResourceListBody>
    </ResourceListContent>
  );
}

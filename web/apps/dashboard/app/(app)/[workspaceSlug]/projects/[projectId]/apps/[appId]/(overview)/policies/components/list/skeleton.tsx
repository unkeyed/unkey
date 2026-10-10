import { ResourceListBody, ResourceListContent, ResourceListItem, Skeleton } from "@unkey/ui";
import { PoliciesListHeader } from "./row";

export function PoliciesListSkeleton() {
  return (
    <ResourceListContent aria-busy="true">
      <PoliciesListHeader />
      <ResourceListBody aria-hidden="true">
        {Array.from({ length: 3 }, (_, i) => `skeleton-${i}`).map((key) => (
          <ResourceListItem
            key={key}
            className="grid h-12 grid-cols-[56px_minmax(0,1.4fr)_minmax(0,1fr)_minmax(0,1fr)_32px] items-center gap-4 px-4"
          >
            <Skeleton className="size-6 rounded-md" />
            <Skeleton className="h-3 w-32" />
            <Skeleton className="h-3 w-20" />
            <Skeleton className="h-3 w-56" />
            <span />
          </ResourceListItem>
        ))}
      </ResourceListBody>
    </ResourceListContent>
  );
}

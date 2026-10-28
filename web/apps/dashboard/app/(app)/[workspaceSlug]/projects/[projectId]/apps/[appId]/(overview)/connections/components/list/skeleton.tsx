import { ResourceListBody, ResourceListContent, ResourceListItem, Skeleton } from "@unkey/ui";
import { ConnectionsListHeader } from "./row";

export function ConnectionsListSkeleton() {
  return (
    <ResourceListContent aria-busy="true">
      <ConnectionsListHeader />
      <ResourceListBody aria-hidden="true">
        {Array.from({ length: 3 }, (_, i) => `skeleton-${i}`).map((key) => (
          <ResourceListItem
            key={key}
            className="grid h-12 grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)_minmax(0,1.2fr)_32px] items-center gap-4 px-4"
          >
            <Skeleton className="h-3 w-32" />
            <Skeleton className="h-3 w-24" />
            <Skeleton className="h-3 w-48" />
            <span />
          </ResourceListItem>
        ))}
      </ResourceListBody>
    </ResourceListContent>
  );
}

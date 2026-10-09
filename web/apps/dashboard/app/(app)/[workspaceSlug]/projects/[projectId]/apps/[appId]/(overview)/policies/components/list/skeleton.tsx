import { ResourceListBody, ResourceListContent, ResourceListItem, Skeleton } from "@unkey/ui";
import { cn } from "cn";
import { POLICY_COLUMNS, PoliciesListHeader } from "./row";

export function PoliciesListSkeleton() {
  return (
    <ResourceListContent aria-busy="true">
      <PoliciesListHeader />
      <ResourceListBody aria-hidden="true">
        {Array.from({ length: 3 }, (_, i) => `skeleton-${i}`).map((key) => (
          <ResourceListItem key={key} className={cn(POLICY_COLUMNS, "h-12 px-4")}>
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

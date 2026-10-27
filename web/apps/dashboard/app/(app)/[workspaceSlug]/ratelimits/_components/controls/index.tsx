import { useFilters } from "@/app/(app)/[workspaceSlug]/ratelimits/[namespaceId]/_overview/hooks/use-filters";
import { ResourceSearchInput } from "@/components/resource-search-input";
import { TimeRangeFilter } from "@/components/time-range-filter";
import { ResourceListHeader, type ResourceListView, ResourceListViewToggle } from "@unkey/ui";

export function NamespaceListControls({
  view,
  onViewChange,
}: {
  view: ResourceListView;
  onViewChange: (view: ResourceListView) => void;
}) {
  const { filters, updateFilters } = useFilters();
  return (
    <ResourceListHeader>
      <ResourceSearchInput label="Search namespaces" placeholder="Search namespaces" />
      <TimeRangeFilter filters={filters} updateFilters={updateFilters} look="list" />
      <ResourceListViewToggle value={view} onValueChange={onViewChange} />
    </ResourceListHeader>
  );
}

import { ResourceSearchInput } from "@/components/resource-search-input";
import { TimeRangeFilter } from "@/components/time-range-filter";
import { ResourceListHeader, type ResourceListView, ResourceListViewToggle } from "@unkey/ui";
import { useFilters } from "../hooks/use-filters";

export function ApiListControls({
  view,
  onViewChange,
}: {
  view: ResourceListView;
  onViewChange: (view: ResourceListView) => void;
}) {
  const { filters, updateFilters } = useFilters();
  return (
    <ResourceListHeader>
      <ResourceSearchInput label="Search keyspaces" placeholder="Search keyspaces by name or ID" />
      <TimeRangeFilter filters={filters} updateFilters={updateFilters} look="list" />
      <ResourceListViewToggle value={view} onValueChange={onViewChange} />
    </ResourceListHeader>
  );
}

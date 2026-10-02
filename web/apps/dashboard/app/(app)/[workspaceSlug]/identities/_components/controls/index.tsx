import { ResourceSearchInput } from "@/components/resource-search-input";
import { ResourceListHeader } from "@unkey/ui";

export function IdentitiesListControls() {
  return (
    <ResourceListHeader>
      <ResourceSearchInput
        queryKey="search"
        label="Search identities"
        placeholder="Search identities by ID or external ID..."
      />
    </ResourceListHeader>
  );
}

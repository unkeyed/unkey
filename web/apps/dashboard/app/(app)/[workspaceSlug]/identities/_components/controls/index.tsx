import { ResourceListHeader } from "@unkey/ui";
import { IdentitiesSearch } from "./identities-search";

export function IdentitiesListControls() {
  return (
    <ResourceListHeader>
      <IdentitiesSearch />
    </ResourceListHeader>
  );
}

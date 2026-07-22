"use client";

import { ResourceList } from "@unkey/ui";
import { IdentitiesListControls } from "./controls";
import { IdentitiesList } from "./table/identities-list";

export const IdentitiesClient = () => {
  return (
    <ResourceList>
      <IdentitiesListControls />
      <IdentitiesList />
    </ResourceList>
  );
};

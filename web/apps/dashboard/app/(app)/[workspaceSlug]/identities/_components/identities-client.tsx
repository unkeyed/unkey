"use client";

import { IdentitiesListControlCloud } from "./control-cloud";
import { IdentitiesListControls } from "./controls";
import { IdentitiesList } from "./table/identities-list";

export const IdentitiesClient = () => {
  return (
    <div className="flex flex-col">
      <IdentitiesListControls />
      <IdentitiesListControlCloud />
      <IdentitiesList />
    </div>
  );
};

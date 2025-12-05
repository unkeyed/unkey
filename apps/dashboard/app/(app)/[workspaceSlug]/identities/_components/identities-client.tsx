"use client";

import { IdentitiesListControls } from "./controls";
import { IdentitiesList } from "./table/identities-list";

export const IdentitiesClient = () => {
  return (
    <div className="flex flex-col">
      <IdentitiesListControls />
      <IdentitiesList />
    </div>
  );
};

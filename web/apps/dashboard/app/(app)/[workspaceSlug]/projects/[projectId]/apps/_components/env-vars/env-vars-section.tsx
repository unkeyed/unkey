"use client";

import { useState } from "react";
import { useAppId, useProjectData } from "../../[appId]/(overview)/data-provider";
import { ALL_ENVIRONMENTS } from "../environment-scope";
import { EnvVarsList } from "./list/env-vars-list";
import { EnvVarsToolbar, type SortOption } from "./toolbar/env-vars-toolbar";

export function EnvVarsSection() {
  const { projectId, environments } = useProjectData();
  const appId = useAppId();
  const [environmentFilter, setEnvironmentFilter] = useState(ALL_ENVIRONMENTS);
  const [sortBy, setSortBy] = useState<SortOption>("last-updated");

  return (
    <>
      <EnvVarsToolbar
        environmentFilter={environmentFilter}
        onEnvironmentFilterChange={setEnvironmentFilter}
        sortBy={sortBy}
        onSortChange={setSortBy}
      />
      <EnvVarsList
        projectId={projectId}
        appId={appId}
        environments={environments}
        environmentFilter={environmentFilter}
        sortBy={sortBy}
      />
    </>
  );
}

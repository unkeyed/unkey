"use client";

import { useAppScope, useEnvironments } from "@/lib/deploy-data";
import { useState } from "react";
import { ALL_ENVIRONMENTS } from "../environment-scope";
import { EnvVarsList } from "./list/env-vars-list";
import { EnvVarsToolbar, type SortOption } from "./toolbar/env-vars-toolbar";

export function EnvVarsSection() {
  const { projectId, appId } = useAppScope();
  const { data: environments } = useEnvironments();
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

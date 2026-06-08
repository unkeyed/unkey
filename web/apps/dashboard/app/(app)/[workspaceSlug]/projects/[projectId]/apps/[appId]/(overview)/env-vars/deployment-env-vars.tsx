"use client";

import { useState } from "react";
import { useProjectData } from "../data-provider";
import { useOptionalProjectLayout } from "../layout-provider";
import { AddEnvVarExpandable } from "./components/add/add-env-var-expandable";
import { EnvVarsList } from "./components/list/env-vars-list";
import { EnvVarsHeader } from "./components/toolbar/env-vars-header";
import {
  EnvVarsToolbar,
  type EnvironmentFilter,
  type SortOption,
} from "./components/toolbar/env-vars-toolbar";

export function DeploymentEnvVars() {
  const { appId, environments } = useProjectData();
  const layout = useOptionalProjectLayout();
  const [searchQuery, setSearchQuery] = useState("");
  const [environmentFilter, setEnvironmentFilter] = useState<EnvironmentFilter>("all");
  const [sortBy, setSortBy] = useState<SortOption>("last-updated");
  const [isAddOpen, setIsAddOpen] = useState(false);

  if (!appId) {
    return null;
  }

  return (
    <>
      <div className="flex flex-col gap-5">
        <EnvVarsHeader isAddOpen={isAddOpen} onToggleAdd={() => setIsAddOpen((prev) => !prev)} />
        <AddEnvVarExpandable
          appId={appId}
          // NOTE: If we are in the onboarding this can start from top of the page
          tableDistanceToTop={layout?.tableDistanceToTop ?? 0}
          isOpen={isAddOpen}
          onClose={() => setIsAddOpen(false)}
        />
        <EnvVarsToolbar
          searchQuery={searchQuery}
          onSearchChange={setSearchQuery}
          environmentFilter={environmentFilter}
          onEnvironmentFilterChange={setEnvironmentFilter}
          environments={environments}
          sortBy={sortBy}
          onSortChange={setSortBy}
        />
        <EnvVarsList
          appId={appId}
          environments={environments}
          searchQuery={searchQuery}
          environmentFilter={environmentFilter}
          sortBy={sortBy}
        />
      </div>
    </>
  );
}

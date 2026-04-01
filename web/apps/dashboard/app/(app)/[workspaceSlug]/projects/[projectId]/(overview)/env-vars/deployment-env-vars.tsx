"use client";

import { useState } from "react";
import { useProjectData } from "../data-provider";
import { useOptionalProjectLayout } from "../layout-provider";
import { PendingRedeployBanner } from "../settings/pending-redeploy-banner";
import { AddEnvVarExpandable } from "./components/add-env-var-expandable";
import { EnvVarsHeader } from "./components/env-vars-header";
import { EnvVarsList } from "./components/env-vars-list";
import {
  EnvVarsToolbar,
  type EnvironmentFilter,
  type SortOption,
} from "./components/env-vars-toolbar";

export function DeploymentEnvVars() {
  const { projectId, environments } = useProjectData();
  const layout = useOptionalProjectLayout();
  const [searchQuery, setSearchQuery] = useState("");
  const [environmentFilter, setEnvironmentFilter] = useState<EnvironmentFilter>("all");
  const [sortBy, setSortBy] = useState<SortOption>("last-updated");
  const [isAddOpen, setIsAddOpen] = useState(false);

  return (
    <>
      <div className="flex flex-col gap-5">
        <EnvVarsHeader isAddOpen={isAddOpen} onToggleAdd={() => setIsAddOpen((prev) => !prev)} />
        <AddEnvVarExpandable
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
          projectId={projectId}
          environments={environments}
          searchQuery={searchQuery}
          environmentFilter={environmentFilter}
          sortBy={sortBy}
        />
      </div>
      <PendingRedeployBanner />
    </>
  );
}

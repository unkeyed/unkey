"use client";

import { useState } from "react";
import { useProjectData } from "../data-provider";
import { AddEnvVarExpandable } from "./components/add/add-env-var-expandable";
import { EnvVarsList } from "./components/list/env-vars-list";
import { EnvVarsHeader } from "./components/toolbar/env-vars-header";
import {
  EnvVarsToolbar,
  type EnvironmentFilter,
  type SortOption,
} from "./components/toolbar/env-vars-toolbar";

type EnvVarsBodyProps = {
  isAddOpen: boolean;
  onCloseAdd: () => void;
  panelTopOffset: number;
};

export function EnvVarsBody({ isAddOpen, onCloseAdd, panelTopOffset }: EnvVarsBodyProps) {
  const { appId, environments } = useProjectData();
  const [searchQuery, setSearchQuery] = useState("");
  const [environmentFilter, setEnvironmentFilter] = useState<EnvironmentFilter>("all");
  const [sortBy, setSortBy] = useState<SortOption>("last-updated");

  if (!appId) {
    return null;
  }

  return (
    <>
      <AddEnvVarExpandable
        appId={appId}
        tableDistanceToTop={panelTopOffset}
        isOpen={isAddOpen}
        onClose={onCloseAdd}
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
    </>
  );
}

export function DeploymentEnvVars() {
  const { appId } = useProjectData();
  const [isAddOpen, setIsAddOpen] = useState(false);

  if (!appId) {
    return null;
  }

  return (
    <div className="flex flex-col gap-5">
      <EnvVarsHeader isAddOpen={isAddOpen} onToggleAdd={() => setIsAddOpen((prev) => !prev)} />
      <EnvVarsBody
        isAddOpen={isAddOpen}
        onCloseAdd={() => setIsAddOpen(false)}
        panelTopOffset={0}
      />
    </div>
  );
}

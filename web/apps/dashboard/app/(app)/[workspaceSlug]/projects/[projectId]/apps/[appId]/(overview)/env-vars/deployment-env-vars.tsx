"use client";

import { useState } from "react";
import { useProjectData } from "../data-provider";
import { AddEnvVarExpandable } from "./components/add/add-env-var-expandable";
import { EnvVarsList } from "./components/list/env-vars-list";
import {
  EnvVarsToolbar,
  type EnvironmentFilter,
  type SortOption,
} from "./components/toolbar/env-vars-toolbar";

type EnvVarsBodyProps = {
  isAddOpen: boolean;
  onCloseAdd: () => void;
};

export function EnvVarsBody({ isAddOpen, onCloseAdd }: EnvVarsBodyProps) {
  const { projectId, appId, environments } = useProjectData();
  const [searchQuery, setSearchQuery] = useState("");
  const [environmentFilter, setEnvironmentFilter] = useState<EnvironmentFilter>("all");
  const [sortBy, setSortBy] = useState<SortOption>("last-updated");

  if (!appId) {
    return null;
  }

  return (
    <>
      <AddEnvVarExpandable
        projectId={projectId}
        appId={appId}
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
        projectId={projectId}
        appId={appId}
        environments={environments}
        searchQuery={searchQuery}
        environmentFilter={environmentFilter}
        sortBy={sortBy}
      />
    </>
  );
}

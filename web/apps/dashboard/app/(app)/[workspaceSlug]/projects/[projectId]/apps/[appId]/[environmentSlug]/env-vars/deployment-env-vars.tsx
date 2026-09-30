"use client";

import { useState } from "react";
import { useProjectData } from "../../data-provider";
import { AddEnvVarExpandable } from "./components/add/add-env-var-expandable";
import { EnvVarsList } from "./components/list/env-vars-list";
import { EnvVarsHeader } from "./components/toolbar/env-vars-header";
import { EnvVarsToolbar, type SortOption } from "./components/toolbar/env-vars-toolbar";
import type { EnvVarsScope } from "./scope";

type EnvVarsBodyProps = {
  scope: EnvVarsScope;
  isAddOpen: boolean;
  onCloseAdd: () => void;
};

export function EnvVarsBody({ scope, isAddOpen, onCloseAdd }: EnvVarsBodyProps) {
  const { projectId, appId, environments } = useProjectData();
  const [searchQuery, setSearchQuery] = useState("");
  const [sortBy, setSortBy] = useState<SortOption>("last-updated");

  if (!appId) {
    return null;
  }

  return (
    <>
      <AddEnvVarExpandable
        projectId={projectId}
        appId={appId}
        scope={scope}
        isOpen={isAddOpen}
        onClose={onCloseAdd}
      />
      <EnvVarsToolbar
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        sortBy={sortBy}
        onSortChange={setSortBy}
      />
      <EnvVarsList
        projectId={projectId}
        appId={appId}
        environments={environments}
        scope={scope}
        searchQuery={searchQuery}
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
        scope={{ kind: "all" }}
        isAddOpen={isAddOpen}
        onCloseAdd={() => setIsAddOpen(false)}
      />
    </div>
  );
}

"use client";

import { useState } from "react";
import { ProjectContentWrapper } from "../../components/project-content-wrapper";
import { AddEnvVarExpandable } from "./components/add-env-var-expandable";
import { EnvVarsHeader } from "./components/env-vars-header";
import { EnvVarsList } from "./components/env-vars-list";
import { type EnvironmentFilter, type SortOption, EnvVarsToolbar } from "./components/env-vars-toolbar";
import { useProjectData } from "../data-provider";
import { useProjectLayout } from "../layout-provider";

export default function EnvVarsPage() {
  const { projectId, environments } = useProjectData();
  const { tableDistanceToTop } = useProjectLayout();
  const [searchQuery, setSearchQuery] = useState("");
  const [environmentFilter, setEnvironmentFilter] = useState<EnvironmentFilter>("all");
  const [sortBy, setSortBy] = useState<SortOption>("last-updated");
  const [isAddOpen, setIsAddOpen] = useState(false);

  return (
    <ProjectContentWrapper centered maxWidth="960px" className="mt-8">
      <div className="flex flex-col gap-5">
        <EnvVarsHeader
          isAddOpen={isAddOpen}
          onToggleAdd={() => setIsAddOpen((prev) => !prev)}
        />
        <AddEnvVarExpandable
          tableDistanceToTop={tableDistanceToTop}
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
    </ProjectContentWrapper>
  );
}

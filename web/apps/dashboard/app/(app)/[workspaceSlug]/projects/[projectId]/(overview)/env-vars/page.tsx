"use client";

import { useState } from "react";
import { ProjectContentWrapper } from "../../components/project-content-wrapper";
import { EnvVarsHeader } from "./components/env-vars-header";
import { EnvVarsList } from "./components/env-vars-list";
import { type EnvironmentFilter, type SortOption, EnvVarsToolbar } from "./components/env-vars-toolbar";
import { useProjectData } from "../data-provider";

export default function EnvVarsPage() {
  const { projectId, environments } = useProjectData();
  const [searchQuery, setSearchQuery] = useState("");
  const [environmentFilter, setEnvironmentFilter] = useState<EnvironmentFilter>("all");
  const [sortBy, setSortBy] = useState<SortOption>("last-updated");

  return (
    <ProjectContentWrapper centered maxWidth="960px" className="mt-8">
      <div className="flex flex-col gap-5">
        <EnvVarsHeader />
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

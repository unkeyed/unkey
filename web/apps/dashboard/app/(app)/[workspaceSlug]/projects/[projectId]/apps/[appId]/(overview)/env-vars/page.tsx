"use client";

import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import { useState } from "react";
import { ALL_ENVIRONMENTS } from "../../../_components/env-vars/shared/environment-select";
import { useAppId, useProjectData } from "../data-provider";
import { AddEnvVarsButton } from "./components/add/add-env-vars-button";
import { EnvVarsList } from "./components/list/env-vars-list";
import { EnvVarsToolbar, type SortOption } from "./components/toolbar/env-vars-toolbar";

export default function EnvVarsPage() {
  const { projectId, environments } = useProjectData();
  const appId = useAppId();
  const [environmentFilter, setEnvironmentFilter] = useState(ALL_ENVIRONMENTS);
  const [sortBy, setSortBy] = useState<SortOption>("last-updated");

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Environment variables</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <AddEnvVarsButton />
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
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
      </PageBody>
    </PageContainer>
  );
}

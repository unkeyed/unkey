"use client";

import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
} from "@unkey/ui";
import { AppGroupList } from "./app-group-list";
import { EnvVarOverviewToolbar } from "./env-var-overview-toolbar";
import { useEnvVarOverview } from "./use-env-var-overview";

export default function EnvVarsOverviewPage() {
  const view = useEnvVarOverview();

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Environment Variables</PageHeaderTitle>
          <PageHeaderDescription>
            Every variable in this workspace, grouped by app.
          </PageHeaderDescription>
        </PageHeaderContent>
      </PageHeader>
      <PageBody className="gap-4">
        <EnvVarOverviewToolbar toolbar={view.toolbar} onChange={view.setToolbar} />
        <AppGroupList view={view} />
      </PageBody>
    </PageContainer>
  );
}

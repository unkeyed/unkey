"use client";

import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import { AddEnvVarsButton } from "../../../_components/env-vars/add/add-env-vars-button";
import { EnvVarsSection } from "../../../_components/env-vars/env-vars-section";

export default function EnvVarsPage() {
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
        <EnvVarsSection />
      </PageBody>
    </PageContainer>
  );
}

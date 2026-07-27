"use client";

import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import { CreateNamespaceButton } from "./_components/create-namespace-button";
import { NamespaceListClient } from "./_components/namespace-list-client";

export default function RatelimitOverviewPage() {
  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Ratelimits</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <CreateNamespaceButton />
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <NamespaceListClient />
      </PageBody>
    </PageContainer>
  );
}

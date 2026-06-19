"use client";

import { PageBody, PageContainer, PageHeader, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";

export default function Overview() {
  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Overview</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <PageBody />
    </PageContainer>
  );
}

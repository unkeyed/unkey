"use client";

import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
  ResourceList,
} from "@unkey/ui";
import { AlertsList } from "./alerts-list";

export default function AlertsPage() {
  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Alerts</PageHeaderTitle>
          <PageHeaderDescription>
            Production anomalies detected against each app's recent baseline.
          </PageHeaderDescription>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>
        <ResourceList>
          <AlertsList />
        </ResourceList>
      </PageBody>
    </PageContainer>
  );
}

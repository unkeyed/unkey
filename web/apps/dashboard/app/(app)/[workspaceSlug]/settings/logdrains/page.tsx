"use client";

import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
  ResourceList,
} from "@unkey/ui";
import { CreateLogdrainButton } from "./create-logdrain-button";
import { LogdrainsList } from "./logdrains-list";

export default function LogdrainsPage() {
  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Log Drains</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <CreateLogdrainButton />
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <ResourceList>
          <LogdrainsList />
        </ResourceList>
      </PageBody>
    </PageContainer>
  );
}

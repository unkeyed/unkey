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
import { CreateLogdrainButton } from "./create-logdrain-button";
import { CreateLogdrainPanel } from "./create-logdrain-panel";
import { LogdrainsList } from "./logdrains-list";

export default function LogdrainsPage() {
  const [isCreateOpen, setIsCreateOpen] = useState(false);

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Log Drains</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <CreateLogdrainButton onClick={() => setIsCreateOpen(true)} />
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <LogdrainsList onCreate={() => setIsCreateOpen(true)} />
      </PageBody>

      <CreateLogdrainPanel isOpen={isCreateOpen} onClose={() => setIsCreateOpen(false)} />
    </PageContainer>
  );
}

"use client";

import { trpc } from "@/lib/trpc/client";
import { IconPlusOutline18 } from "@unkey/icons";
import {
  Button,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import { useState } from "react";
import { OIDCVercelIntegrationPanel } from "./oidc-vercel-integration-panel";
import { OIDCVercelIntegrationsList } from "./oidc-vercel-integrations-list";

export default function OIDCVercelPage() {
  const [createOpen, setCreateOpen] = useState(false);
  const integrations = trpc.oidcVercel.list.useQuery();

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Vercel OIDC</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <Button
            size="md"
            variant="primary"
            onClick={() => setCreateOpen(true)}
            disabled={integrations.isLoading || integrations.isError}
          >
            <IconPlusOutline18 />
            Add Integration
          </Button>
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <OIDCVercelIntegrationsList query={integrations} onCreate={() => setCreateOpen(true)} />
      </PageBody>
      <OIDCVercelIntegrationPanel isOpen={createOpen} onClose={() => setCreateOpen(false)} />
    </PageContainer>
  );
}

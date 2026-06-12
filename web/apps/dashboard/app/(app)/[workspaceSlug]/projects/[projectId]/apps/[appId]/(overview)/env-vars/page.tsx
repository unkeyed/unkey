"use client";

import { TOP_NAV_HEIGHT } from "@/components/navigation/top-nav";
import { Plus } from "@unkey/icons";
import {
  Button,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
} from "@unkey/ui";
import { useState } from "react";
import { EnvVarsBody } from "./deployment-env-vars";

export default function EnvVarsPage() {
  const [isAddOpen, setIsAddOpen] = useState(false);

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Environment Variables</PageHeaderTitle>
          <PageHeaderDescription>
            Store API keys, tokens, and config securely. Changes apply on next deploy.
          </PageHeaderDescription>
        </PageHeaderContent>
        <PageHeaderActions>
          <Button
            size="md"
            onClick={() => setIsAddOpen((prev) => !prev)}
            variant={isAddOpen ? "outline" : "primary"}
          >
            <Plus iconSize="sm-regular" />
            Add environment variable
          </Button>
        </PageHeaderActions>
      </PageHeader>
      <PageBody className="flex flex-col gap-5 pt-6 pb-20">
        <EnvVarsBody
          isAddOpen={isAddOpen}
          onCloseAdd={() => setIsAddOpen(false)}
          panelTopOffset={TOP_NAV_HEIGHT}
        />
      </PageBody>
    </PageContainer>
  );
}

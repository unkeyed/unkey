"use client";

import { IconPlusOutline18 } from "@unkey/icons";
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
import { useAppEnvironment } from "../environment-context";
import { EnvVarsBody } from "./deployment-env-vars";

export default function EnvVarsPage() {
  const [isAddOpen, setIsAddOpen] = useState(false);
  const { environment } = useAppEnvironment();

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
            <IconPlusOutline18 />
            Add environment variable
          </Button>
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <EnvVarsBody
          scope={{ kind: "environment", environmentId: environment.id }}
          isAddOpen={isAddOpen}
          onCloseAdd={() => setIsAddOpen(false)}
        />
      </PageBody>
    </PageContainer>
  );
}

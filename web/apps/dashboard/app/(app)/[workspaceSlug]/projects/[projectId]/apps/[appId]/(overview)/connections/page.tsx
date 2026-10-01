"use client";

import { PageBody, PageContainer, Skeleton } from "@unkey/ui";
import { useState } from "react";
import { useAppId, useProjectData } from "../data-provider";
import { ConnectionCanvas, ConnectionsHeader } from "./connection-canvas";
import { isProduction } from "./connection-rules";

export default function ConnectionsPage() {
  const { projectId, environments, isEnvironmentsLoading } = useProjectData();
  const appId = useAppId();
  const [selectedEnvironment, setSelectedEnvironment] = useState<string | null>(null);
  const environment =
    environments.find((env) => env.id === selectedEnvironment) ??
    environments.find(isProduction) ??
    environments[0];

  return (
    <PageContainer>
      {isEnvironmentsLoading || !environment ? (
        <>
          <ConnectionsHeader />
          <PageBody>
            {isEnvironmentsLoading ? (
              <Skeleton className="h-96 w-full" />
            ) : (
              <p className="text-sm text-gray-9">
                Create an environment before adding connections.
              </p>
            )}
          </PageBody>
        </>
      ) : (
        <ConnectionCanvas
          key={environment.id}
          projectId={projectId}
          appId={appId}
          environment={environment}
          environments={environments}
          onEnvironmentChange={setSelectedEnvironment}
        />
      )}
    </PageContainer>
  );
}

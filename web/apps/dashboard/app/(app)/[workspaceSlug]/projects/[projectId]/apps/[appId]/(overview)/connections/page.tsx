"use client";

import { useAppScope, useEnvironments } from "@/lib/deploy-data";
import { useConnectionsEnabled } from "@/lib/flags/use-connections-enabled";
import { PageBody, PageContainer, Skeleton } from "@unkey/ui";
import { notFound } from "next/navigation";
import { useState } from "react";
import { ConnectionCanvas, ConnectionsHeader } from "./connection-canvas";
import { isProduction } from "./connection-rules";

export default function ConnectionsPage() {
  const enabled = useConnectionsEnabled();
  if (!enabled) {
    notFound();
  }
  return <ConnectionsContent />;
}

function ConnectionsContent() {
  const { projectId, appId } = useAppScope();
  const { data: environments = [], isLoading: isEnvironmentsLoading } = useEnvironments();
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

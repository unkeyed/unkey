"use client";

import { PageLoading } from "@/components/dashboard/page-loading";
import { useBillingUIUpgrades } from "@/lib/flags/use-billing-ui-upgrades";
import { trpc } from "@/lib/trpc/client";
import {
  Empty,
  ItemGroup,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import { notFound } from "next/navigation";
import { useMemo } from "react";
import { MetricsExplorer } from "./metrics-explorer";

export default function MetricsPage() {
  const enabled = useBillingUIUpgrades();
  const projects = trpc.deploy.project.list.useQuery(undefined, { enabled, retry: 1 });
  const environments = trpc.deploy.environment.listAll.useQuery(undefined, { enabled, retry: 1 });
  const scope = useMemo(() => {
    const environmentsByApp = Map.groupBy(
      environments.data ?? [],
      (environment) => environment.appId,
    );
    return {
      projects: (projects.data ?? []).map((project) => ({
        projectId: project.id,
        name: project.name,
        apps: project.apps.map((app) => ({
          appId: app.id,
          name: app.name,
          environments: (environmentsByApp.get(app.id) ?? []).map((environment) => ({
            environmentId: environment.id,
            name: environment.name,
          })),
        })),
      })),
    };
  }, [environments.data, projects.data]);

  if (!enabled) {
    notFound();
  }

  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Metrics</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>
        {projects.isLoading || environments.isLoading ? (
          <PageLoading message="Loading metrics..." />
        ) : projects.isError || environments.isError ? (
          <ItemGroup>
            <div className="px-4 py-12">
              <Empty className="w-full">
                <Empty.Title>Metrics unavailable</Empty.Title>
                <Empty.Description>
                  We could not load the compute scope. Please try again later.
                </Empty.Description>
              </Empty>
            </div>
          </ItemGroup>
        ) : (
          <ItemGroup>
            <MetricsExplorer scope={scope} />
          </ItemGroup>
        )}
      </PageBody>
    </PageContainer>
  );
}

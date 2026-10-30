"use client";

import { ListPager } from "@/components/list-pager";
import { LoadError } from "@/components/load-error";
import { useConsumeSearchFlag } from "@/hooks/use-consume-search-flag";
import { IconLinkOutline18, IconPlusOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import {
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
  ResourceListContent,
} from "@unkey/ui";
import { useCallback } from "react";
import { useAppId, useProjectData } from "../data-provider";
import { useSlidePanels } from "../hooks/use-slide-panels";
import { ConnectionPanel } from "./components/connection-form/connection-panel";
import { ConnectionsList } from "./components/list";
import { RetainedConnections } from "./components/list/retained";
import { ConnectionsListSkeleton } from "./components/list/skeleton";
import { removalOf, useConnectionMutation } from "./hooks/use-connection-actions";
import { useConnectionsData } from "./hooks/use-connections-data";
import { HowConnectionsWorkPanel } from "./how-connections-work";
import { type Environment, type ListedConnection, isProduction } from "./model/target";

type PanelState = { type: "connection"; before: ListedConnection | null } | { type: "guide" };

export default function ConnectionsPage() {
  const { projectId, environments, isEnvironmentsLoading } = useProjectData();
  const appId = useAppId();
  const environment = environments.find(isProduction);

  if (!environment) {
    return (
      <PageContainer>
        <PageHeader>
          <PageHeaderContent>
            <PageHeaderTitle>Connections</PageHeaderTitle>
          </PageHeaderContent>
        </PageHeader>
        <PageBody>
          {isEnvironmentsLoading ? (
            <ConnectionsListSkeleton />
          ) : (
            <p className="text-sm text-gray-9">
              Create a production environment before adding connections.
            </p>
          )}
        </PageBody>
      </PageContainer>
    );
  }

  return <ProductionConnections projectId={projectId} appId={appId} environment={environment} />;
}

function ProductionConnections({
  projectId,
  appId,
  environment,
}: {
  projectId: string;
  appId: string;
  environment: Environment;
}) {
  const environmentId = environment.id;
  const data = useConnectionsData(projectId, appId, environmentId);
  const mutation = useConnectionMutation({ projectId, appId, environmentId });
  const panels = useSlidePanels<PanelState>();
  const openAdd = useCallback(
    () => panels.open({ type: "connection", before: null }),
    [panels.open],
  );
  useConsumeSearchFlag(
    "add",
    data.snapshot.status !== "loading",
    data.snapshot.status === "ready" ? openAdd : null,
  );

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Connections</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <Button size="md" variant="outline" onClick={() => panels.open({ type: "guide" })}>
            How connections work
          </Button>
          <Button
            size="md"
            variant="primary"
            disabled={data.snapshot.status !== "ready"}
            onClick={openAdd}
          >
            <IconPlusOutline18 />
            Add connection
          </Button>
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        {match(data.view)
          .with({ status: "failed" }, ({ error, retry }) => (
            <LoadError title="Could not load connections" error={error} onRetry={retry} />
          ))
          .with({ status: "loading" }, () => <ConnectionsListSkeleton />)
          .with({ status: "empty" }, () => (
            <EmptyState>
              <EmptyStateIcon>
                <IconLinkOutline18 />
              </EmptyStateIcon>
              <EmptyStateHeader>
                <EmptyStateTitle>No connections yet</EmptyStateTitle>
                <EmptyStateDescription>
                  Connections let this app call other apps in the project over the private network.
                </EmptyStateDescription>
              </EmptyStateHeader>
              <EmptyStateActions>
                <Button variant="primary" size="md" onClick={openAdd}>
                  <IconPlusOutline18 />
                  Add connection
                </Button>
              </EmptyStateActions>
            </EmptyState>
          ))
          .with({ status: "ready" }, ({ rows, footer }) => (
            <ResourceListContent>
              <ConnectionsList
                rows={rows}
                onEdit={(connection) => panels.open({ type: "connection", before: connection })}
                onRemove={(connection) => mutation.run(removalOf(connection))}
                removing={mutation.pending !== null}
              />
              <ListPager state={footer} noun="connections" />
            </ResourceListContent>
          ))
          .exhaustive()}
        <RetainedConnections retained={data.retained} />
        {panels.panel &&
          match(panels.panel)
            .with({ type: "guide" }, () => (
              <HowConnectionsWorkPanel isOpen={panels.isOpen} onClose={panels.close} />
            ))
            .with({ type: "connection" }, ({ before }) =>
              data.snapshot.status === "ready" ? (
                <ConnectionPanel
                  key={panels.session}
                  projectId={projectId}
                  appId={appId}
                  environmentId={environmentId}
                  snapshot={data.snapshot.data}
                  before={before}
                  mutation={mutation}
                  isOpen={panels.isOpen}
                  onClose={panels.close}
                />
              ) : null,
            )
            .exhaustive()}
      </PageBody>
    </PageContainer>
  );
}

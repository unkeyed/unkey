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
import { RemovedConnectionRows } from "./components/list/retained";
import { ConnectionsListSkeleton } from "./components/list/skeleton";
import { useConnectionMutation } from "./hooks/use-connection-actions";
import { useConnectionsData } from "./hooks/use-connections-data";
import { HowConnectionsWorkPanel } from "./how-connections-work";
import { removalOf } from "./model/plan";
import { type Environment, type ListedConnection, environmentOfKind } from "./model/target";

type PanelState = { type: "connection"; before: ListedConnection | null } | { type: "guide" };

export function ConnectionsPage() {
  const { projectId, environments, isEnvironmentsLoading } = useProjectData();
  const appId = useAppId();
  const production = environmentOfKind(environments, "production");
  const preview = environmentOfKind(environments, "preview");

  if (!production) {
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

  return (
    <AppConnections projectId={projectId} appId={appId} production={production} preview={preview} />
  );
}

function AppConnections({
  projectId,
  appId,
  production,
  preview,
}: {
  projectId: string;
  appId: string;
  production: Environment;
  preview: Environment | null;
}) {
  const data = useConnectionsData({ projectId, appId, production, preview });
  const callers = data.callers.status === "ready" ? data.callers.data : null;
  const mutation = useConnectionMutation({ projectId, appId });
  const panels = useSlidePanels<PanelState>();
  const openAdd = useCallback(
    () => panels.open({ type: "connection", before: null }),
    [panels.open],
  );
  const editConnection = (connection: ListedConnection) =>
    panels.open({ type: "connection", before: connection });
  const removeConnection = (connection: ListedConnection) => {
    if (callers) {
      mutation.run({
        action: "remove",
        ops: removalOf(callers, connection.targetAppId),
      });
    }
  };
  const removeDisabled = mutation.pending !== null || !callers;
  const removed = <RemovedConnectionRows retained={data.retained} />;
  const hasRemoved = data.retained.status !== "empty" && data.retained.status !== "loading";
  useConsumeSearchFlag("add", data.callers.status !== "loading", callers ? openAdd : null);

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
          <Button size="md" variant="primary" disabled={!callers} onClick={openAdd}>
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
          .with({ status: "empty" }, () =>
            hasRemoved ? (
              <ResourceListContent>
                <ConnectionsList
                  rows={[]}
                  onEdit={editConnection}
                  onRemove={removeConnection}
                  removeDisabled={removeDisabled}
                  removed={removed}
                />
              </ResourceListContent>
            ) : (
              <EmptyState>
                <EmptyStateIcon>
                  <IconLinkOutline18 />
                </EmptyStateIcon>
                <EmptyStateHeader>
                  <EmptyStateTitle>No connections yet</EmptyStateTitle>
                  <EmptyStateDescription>
                    Connections let this app call other apps in the project over the private
                    network.
                  </EmptyStateDescription>
                </EmptyStateHeader>
                <EmptyStateActions>
                  <Button variant="primary" size="md" onClick={openAdd}>
                    <IconPlusOutline18 />
                    Add connection
                  </Button>
                </EmptyStateActions>
              </EmptyState>
            ),
          )
          .with({ status: "ready" }, ({ rows, footer }) => (
            <ResourceListContent>
              <ConnectionsList
                rows={rows}
                onEdit={editConnection}
                onRemove={removeConnection}
                removeDisabled={removeDisabled}
                removed={removed}
              />
              <ListPager state={footer} noun="connections" />
            </ResourceListContent>
          ))
          .exhaustive()}
        {panels.panel &&
          match(panels.panel)
            .with({ type: "guide" }, () => (
              <HowConnectionsWorkPanel isOpen={panels.isOpen} onClose={panels.close} />
            ))
            .with({ type: "connection" }, ({ before }) =>
              callers ? (
                <ConnectionPanel
                  key={panels.session}
                  callers={callers}
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

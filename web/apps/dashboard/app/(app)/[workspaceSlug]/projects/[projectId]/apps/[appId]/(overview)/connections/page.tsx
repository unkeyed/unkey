"use client";

import { LoadError } from "@/components/load-error";
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
import { useSlidePanels } from "../hooks/use-slide-panels";
import { AddConnectionPanel } from "./components/connection-form/add-connection-panel";
import { EditConnectionPanel } from "./components/connection-form/edit-connection-panel";
import { ConnectionsList } from "./components/list";
import { RetainedConnections } from "./components/list/retained";
import { ConnectionsListSkeleton } from "./components/list/skeleton";
import { useConnectionActions } from "./hooks/use-connection-actions";
import { useConnectionsData } from "./hooks/use-connections-data";
import { HowConnectionsWorkPanel } from "./how-connections-work";

type ConnectionPanel = { type: "add" } | { type: "edit"; targetAppId: string } | { type: "guide" };

export default function ConnectionsPage() {
  const data = useConnectionsData();
  const { envs, merged, targets, canWrite } = data;
  const actions = useConnectionActions(data);
  const panels = useSlidePanels<ConnectionPanel>();
  const openAdd = () => panels.open({ type: "add" });

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
          <Button size="md" variant="primary" disabled={!canWrite} onClick={openAdd}>
            <IconPlusOutline18 />
            Add connection
          </Button>
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        {match(data.view)
          .with({ type: "error" }, () => (
            <LoadError title="Could not load connections" onRetry={data.retry} />
          ))
          .with({ type: "loading" }, () => <ConnectionsListSkeleton />)
          .with({ type: "empty" }, () => (
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
                <Button variant="primary" size="md" disabled={!canWrite} onClick={openAdd}>
                  <IconPlusOutline18 />
                  Add connection
                </Button>
              </EmptyStateActions>
            </EmptyState>
          ))
          .with({ type: "list" }, () => (
            <ResourceListContent>
              <ConnectionsList
                envs={envs}
                merged={merged}
                targets={targets}
                onEdit={(targetAppId) => panels.open({ type: "edit", targetAppId })}
                onRemove={(connection) => void actions.remove(connection)}
              />
            </ResourceListContent>
          ))
          .exhaustive()}
        <RetainedConnections retained={data.retained} envs={envs} />
        {panels.panel &&
          match(panels.panel)
            .with({ type: "guide" }, () => (
              <HowConnectionsWorkPanel isOpen={panels.isOpen} onClose={panels.close} />
            ))
            .with({ type: "add" }, () => (
              <AddConnectionPanel
                key={panels.session}
                data={data}
                isOpen={panels.isOpen}
                onClose={panels.close}
                onSave={(values, after) => actions.save(null, values, after)}
              />
            ))
            .with({ type: "edit" }, ({ targetAppId }) => (
              <EditConnectionPanel
                key={panels.session}
                data={data}
                row={merged.find((connection) => connection.targetAppId === targetAppId)}
                isOpen={panels.isOpen}
                onClose={panels.close}
                onSave={actions.save}
                onRemove={actions.remove}
              />
            ))
            .exhaustive()}
        {actions.planGate}
      </PageBody>
    </PageContainer>
  );
}

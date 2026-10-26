"use client";
import { LoadError } from "@/components/load-error";
import { useConsumeSearchFlag } from "@/hooks/use-consume-search-flag";
import { IconCircleInfoOutline18, IconPlusOutline18, IconShieldGlobeOutline18 } from "@unkey/icons";
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
import { HowPoliciesWorkPanel } from "./components/how-policies-work";
import { PoliciesList } from "./components/list";
import { PoliciesListSkeleton } from "./components/list/skeleton";
import { AddPolicyPanel } from "./components/policy-form/add-policy-panel";
import { EditPolicyPanel } from "./components/policy-form/edit-policy-panel";
import { usePoliciesData } from "./hooks/use-policies-data";
import { usePolicyActions } from "./hooks/use-policy-actions";
import { usePolicyPanels } from "./hooks/use-policy-panels";
import { otherMatchKeys } from "./model/duplicate-name";

export default function PoliciesPage() {
  const { projectId } = useProjectData();
  const appId = useAppId();
  const data = usePoliciesData();
  const { envs, merged, canWrite } = data;
  const actions = usePolicyActions({ envs, projectId, appId });
  const panels = usePolicyPanels();
  const openAdd = useCallback(() => panels.open({ type: "add" }), [panels.open]);
  useConsumeSearchFlag("add", data.view.type !== "loading", canWrite ? openAdd : null);

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Policies</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <Button size="md" variant="outline" onClick={() => panels.open({ type: "guide" })}>
            How policies work
          </Button>
          <Button size="md" variant="primary" disabled={!canWrite} onClick={openAdd}>
            <IconPlusOutline18 />
            Add policy
          </Button>
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        {match(data.view)
          .with({ type: "error" }, () => (
            <LoadError title="Could not load policies" onRetry={data.retry} />
          ))
          .with({ type: "loading" }, () => <PoliciesListSkeleton />)
          .with({ type: "empty" }, () => (
            <EmptyState>
              <EmptyStateIcon>
                <IconShieldGlobeOutline18 />
              </EmptyStateIcon>
              <EmptyStateHeader>
                <EmptyStateTitle>No policies yet</EmptyStateTitle>
                <EmptyStateDescription>
                  Policies check requests before they reach your app.
                </EmptyStateDescription>
              </EmptyStateHeader>
              <EmptyStateActions>
                <Button variant="primary" size="md" disabled={!canWrite} onClick={openAdd}>
                  <IconPlusOutline18 />
                  Add policy
                </Button>
              </EmptyStateActions>
            </EmptyState>
          ))
          .with({ type: "list" }, () => (
            <div className="flex flex-col gap-3">
              <div className="flex items-center gap-2 rounded-lg bg-grayA-2 px-3 py-2 text-sm text-gray-11">
                <IconCircleInfoOutline18 className="size-3.5 shrink-0" />
                Policies run top to bottom. Drag a row to reorder.
              </div>
              <ResourceListContent>
                <PoliciesList
                  envs={envs}
                  merged={merged}
                  reorderDisabled={actions.replacingList}
                  onReorder={actions.reorder}
                  onDelete={actions.delete}
                  onEdit={(key) => panels.open({ type: "edit", key })}
                />
              </ResourceListContent>
            </div>
          ))
          .exhaustive()}
        {panels.panel &&
          match(panels.panel)
            .with({ type: "guide" }, () => (
              <HowPoliciesWorkPanel isOpen={panels.isOpen} onClose={panels.close} />
            ))
            .with({ type: "add" }, () => (
              <AddPolicyPanel
                key={panels.session}
                envs={envs}
                isOpen={panels.isOpen}
                onClose={panels.close}
                otherMatchKeys={otherMatchKeys(merged, null)}
                onSave={actions.save}
              />
            ))
            .with({ type: "edit" }, ({ key }) => (
              <EditPolicyPanel
                key={panels.session}
                row={merged.find((m) => m.key === key)}
                envs={envs}
                isOpen={panels.isOpen}
                onClose={panels.close}
                otherMatchKeys={otherMatchKeys(merged, key)}
                locked={actions.replacingList}
                onUpdate={actions.update}
                onSetEnabled={actions.setEnabled}
              />
            ))
            .exhaustive()}
      </PageBody>
    </PageContainer>
  );
}

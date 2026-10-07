"use client";
import { policyMatchKey } from "@/lib/collections/deploy/policies.schema";
import { IconCircleInfoOutline18, IconPlusOutline18, IconShieldGlobeOutline18 } from "@unkey/icons";
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
import { useAppId, useProjectData } from "../data-provider";
import { HowPoliciesWorkPanel } from "./components/how-policies-work";
import { PoliciesList } from "./components/list";
import { PoliciesError } from "./components/list/error";
import { PoliciesListSkeleton } from "./components/list/skeleton";
import { AddPolicyPanel } from "./components/policy-form/add-policy-panel";
import { EditPolicyPanel } from "./components/policy-form/edit-policy-panel";
import { usePoliciesData } from "./hooks/use-policies-data";
import { usePolicyActions } from "./hooks/use-policy-actions";
import { usePolicyPanels } from "./hooks/use-policy-panels";

export default function PoliciesPage() {
  const { projectId } = useProjectData();
  const appId = useAppId();
  const data = usePoliciesData();
  const { envs, merged, isLoading, isError, canWrite } = data;
  const actions = usePolicyActions({ ...data, projectId, appId });
  const panels = usePolicyPanels();

  const editingKey = panels.editing?.key;
  const editingRow = merged.find((m) => m.key === editingKey);

  const existingMatchKeys = merged.map((m) => policyMatchKey(m.type, m.name));

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Policies</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <Button size="md" variant="outline" onClick={panels.openGuide}>
            How policies work
          </Button>
          <Button size="md" variant="primary" disabled={!canWrite} onClick={panels.openAdd}>
            <IconPlusOutline18 />
            Add policy
          </Button>
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        {isError ? (
          <PoliciesError />
        ) : isLoading ? (
          <PoliciesListSkeleton />
        ) : merged.length === 0 ? (
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
              <Button variant="primary" size="md" onClick={panels.openAdd}>
                <IconPlusOutline18 />
                Add policy
              </Button>
            </EmptyStateActions>
          </EmptyState>
        ) : (
          <div className="flex flex-col gap-3">
            <div className="flex items-center gap-2 rounded-lg bg-grayA-2 px-3 py-2 text-sm text-gray-11">
              <IconCircleInfoOutline18 className="size-3.5 shrink-0" />
              Policies run top to bottom. Drag a row to reorder.
            </div>
            <ResourceListContent>
              <PoliciesList
                envs={envs}
                merged={merged}
                onReorder={actions.reorder}
                onDelete={actions.delete}
                onEdit={panels.openEdit}
              />
            </ResourceListContent>
          </div>
        )}
        <HowPoliciesWorkPanel isOpen={panels.isGuideOpen} onClose={panels.closeGuide} />
        <AddPolicyPanel
          key={panels.addSession}
          envs={envs}
          isOpen={panels.isAddPanelOpen}
          onClose={panels.closeAdd}
          existingMatchKeys={existingMatchKeys}
          onSave={actions.save}
        />
        <EditPolicyPanel
          editing={panels.editing}
          row={editingRow}
          envs={envs}
          isOpen={panels.isEditPanelOpen}
          onClose={panels.closeEdit}
          existingMatchKeys={existingMatchKeys}
          onUpdate={actions.update}
          onToggleEnv={actions.toggleEnv}
          onAddToEnv={actions.addToEnv}
        />
      </PageBody>
    </PageContainer>
  );
}

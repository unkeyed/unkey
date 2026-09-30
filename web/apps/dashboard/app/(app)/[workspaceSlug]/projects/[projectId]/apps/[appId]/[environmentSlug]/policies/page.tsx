"use client";
import { type Policy, policyMatchKey } from "@/lib/collections/deploy/policies.schema";
import { IconPlusOutline18, IconShieldKeyOutline18 } from "@unkey/icons";
import {
  Button,
  EmptyState,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
} from "@unkey/ui";
import { useAppId, useProjectData } from "../../data-provider";
import { useAppEnvironment } from "../environment-context";
import { PolicyPanel } from "./components/add-panel";
import { PoliciesList } from "./components/list";
import { PoliciesError } from "./components/list/error";
import type { Env, MergedPolicy } from "./components/list/merge";
import { PoliciesListSkeleton } from "./components/list/skeleton";
import { usePoliciesData } from "./hooks/use-policies-data";
import { usePolicyActions } from "./hooks/use-policy-actions";
import { usePolicyPanels } from "./hooks/use-policy-panels";

export default function PoliciesPage() {
  const { projectId } = useProjectData();
  const appId = useAppId();
  const { environment } = useAppEnvironment();
  const env: Env = environment.kind;
  const other: Env = env === "production" ? "preview" : "production";
  const {
    productionId,
    previewId,
    productionSlug,
    previewSlug,
    merged,
    rowsByEnv,
    isLoading,
    isError,
  } = usePoliciesData();
  const otherSlug = other === "production" ? productionSlug : previewSlug;
  const actions = usePolicyActions({ env, productionId, previewId, projectId, appId, rowsByEnv });
  const panels = usePolicyPanels();
  const rows = rowsByEnv[env];

  const editingRow = panels.editing
    ? merged.find((m) => m[env]?.id === panels.editing?.id)
    : undefined;
  const existingMatchKeys = merged.map((m) => policyMatchKey(m.type, m.name));

  // The current environment always receives the policy; the other one only
  // when asked, and an existing copy there is switched off otherwise.
  const save = (policy: Policy, applyToOther: boolean, editing?: MergedPolicy) => {
    const here = { ...policy, enabled: true };
    const there = applyToOther ? { ...policy, enabled: true } : null;
    actions.save(env === "production" ? here : there, env === "production" ? there : here, editing);
  };

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Policies</PageHeaderTitle>
          <PageHeaderDescription>
            Middleware policy chains that protect your API. Policies are evaluated in order, drag to
            reorder.
          </PageHeaderDescription>
        </PageHeaderContent>
        <PageHeaderActions>
          <Button size="md" onClick={panels.openAdd} variant="primary">
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
        ) : rows.length === 0 ? (
          <EmptyState>
            <EmptyStateIcon>
              <IconShieldKeyOutline18 />
            </EmptyStateIcon>
            <EmptyStateHeader>
              <EmptyStateTitle>No policies in {environment.slug}</EmptyStateTitle>
              <EmptyStateDescription>
                Add policies to protect your API with authentication, rate limiting, and more.
                Policies are evaluated sequentially on each incoming request.
              </EmptyStateDescription>
            </EmptyStateHeader>
          </EmptyState>
        ) : (
          <PoliciesList
            rows={rows}
            onToggle={actions.toggle}
            onReorder={actions.reorder}
            onDelete={actions.delete}
            onEdit={panels.openEdit}
          />
        )}
        <PolicyPanel
          mode="add"
          otherEnvironmentSlug={otherSlug}
          isOpen={panels.isAddPanelOpen}
          onClose={panels.closeAdd}
          existingMatchKeys={existingMatchKeys}
          onSave={(policy, applyToOther) => save(policy, applyToOther)}
        />
        {panels.editing !== null && (
          <PolicyPanel
            key={panels.editing.id}
            mode="edit"
            otherEnvironmentSlug={otherSlug}
            isOpen={panels.isEditPanelOpen}
            onClose={panels.closeEdit}
            existingMatchKeys={existingMatchKeys}
            initialPolicy={panels.editing}
            initialApplyToOther={editingRow?.[other]?.enabled ?? false}
            onSave={(policy, applyToOther) => {
              save(policy, applyToOther, editingRow);
              panels.closeEdit();
            }}
          />
        )}
      </PageBody>
    </PageContainer>
  );
}

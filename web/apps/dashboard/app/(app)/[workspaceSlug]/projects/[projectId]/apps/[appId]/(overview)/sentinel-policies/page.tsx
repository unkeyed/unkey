"use client";
import { TOP_NAV_HEIGHT } from "@/components/navigation/top-nav";
import { Plus } from "@unkey/icons";
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
import { SentinelPolicyPanel } from "./components/add-panel";
import { SentinelPoliciesList } from "./components/list";
import { SentinelPoliciesEmpty } from "./components/list/empty";
import { SentinelPoliciesError } from "./components/list/error";
import { SentinelPoliciesListSkeleton } from "./components/list/skeleton";
import { useSentinelPoliciesData } from "./hooks/use-sentinel-policies-data";
import { useSentinelPolicyActions } from "./hooks/use-sentinel-policy-actions";
import { useSentinelPolicyPanels } from "./hooks/use-sentinel-policy-panels";

export default function SentinelPoliciesPage() {
  const { envAId, envBId, envASlug, envBSlug, merged, isLoading, isError } =
    useSentinelPoliciesData();
  const actions = useSentinelPolicyActions({ envAId, envBId });
  const panels = useSentinelPolicyPanels();

  const editingRow = panels.editing ? merged.find((m) => m.id === panels.editing?.id) : undefined;
  const editingEnabled = {
    a: editingRow?.envA?.enabled ?? false,
    b: editingRow?.envB?.enabled ?? false,
  };

  const editingInitialEnvId =
    editingEnabled.a && editingEnabled.b
      ? "__all__"
      : editingEnabled.a
        ? envASlug
        : editingEnabled.b
          ? envBSlug
          : "__all__";

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Sentinel Policies</PageHeaderTitle>
          <PageHeaderDescription>
            Middleware policy chains that protect your API. Policies are evaluated in order, drag to
            reorder.
          </PageHeaderDescription>
        </PageHeaderContent>
        <PageHeaderActions>
          <Button size="md" onClick={panels.openAdd} variant="primary">
            <Plus iconSize="sm-regular" />
            Add Policy
          </Button>
        </PageHeaderActions>
      </PageHeader>
      <PageBody className="flex flex-col gap-5 pt-6 pb-20">
        {isError ? (
          <SentinelPoliciesError />
        ) : isLoading ? (
          <SentinelPoliciesListSkeleton />
        ) : merged.length === 0 ? (
          <SentinelPoliciesEmpty />
        ) : (
          <SentinelPoliciesList
            envASlug={envASlug}
            envBSlug={envBSlug}
            merged={merged}
            onToggleEnv={actions.toggleEnv}
            onAddToEnv={actions.addToEnv}
            onReorder={actions.reorder}
            onDelete={actions.delete}
            onEdit={panels.openEdit}
          />
        )}
        <SentinelPolicyPanel
          mode="add"
          envASlug={envASlug}
          envBSlug={envBSlug}
          isOpen={panels.isAddPanelOpen}
          topOffset={TOP_NAV_HEIGHT}
          onClose={panels.closeAdd}
          onSave={actions.save}
        />
        {panels.editing !== null && (
          <SentinelPolicyPanel
            key={panels.editing.id}
            mode="edit"
            envASlug={envASlug}
            envBSlug={envBSlug}
            isOpen={panels.isEditPanelOpen}
            topOffset={TOP_NAV_HEIGHT}
            onClose={panels.closeEdit}
            initialPolicy={panels.editing}
            initialEnvironmentId={editingInitialEnvId}
            onSave={(prodPolicy, previewPolicy) => {
              actions.save(prodPolicy, previewPolicy);
              panels.closeEdit();
            }}
          />
        )}
      </PageBody>
    </PageContainer>
  );
}

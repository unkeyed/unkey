"use client";
import { policyMatchKey } from "@/lib/collections/deploy/policies.schema";
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
import { IconPlusOutline18 } from "nucleo-ui-outline-18";
import { useAppId, useProjectData } from "../data-provider";
import { PolicyPanel } from "./components/add-panel";
import { PoliciesList } from "./components/list";
import { PoliciesEmpty } from "./components/list/empty";
import { PoliciesError } from "./components/list/error";
import { PoliciesListSkeleton } from "./components/list/skeleton";
import { usePoliciesData } from "./hooks/use-policies-data";
import { usePolicyActions } from "./hooks/use-policy-actions";
import { usePolicyPanels } from "./hooks/use-policy-panels";

export default function PoliciesPage() {
  const { projectId } = useProjectData();
  const appId = useAppId();
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
  const actions = usePolicyActions({
    productionId,
    previewId,
    projectId,
    appId,
    merged,
    rowsByEnv,
  });
  const panels = usePolicyPanels();

  const editingRow = panels.editing
    ? merged.find(
        (m) => m.production?.id === panels.editing?.id || m.preview?.id === panels.editing?.id,
      )
    : undefined;
  const editingEnabled = {
    a: editingRow?.production?.enabled ?? false,
    b: editingRow?.preview?.enabled ?? false,
  };

  const existingMatchKeys = merged.map((m) => policyMatchKey(m.type, m.name));

  const editingInitialEnvId =
    editingEnabled.a && editingEnabled.b
      ? "__all__"
      : editingEnabled.a
        ? productionSlug
        : editingEnabled.b
          ? previewSlug
          : "__all__";

  const editingPolicy =
    editingInitialEnvId === previewSlug
      ? (editingRow?.preview ?? panels.editing)
      : (editingRow?.production ?? panels.editing);

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
        ) : merged.length === 0 ? (
          <PoliciesEmpty />
        ) : (
          <PoliciesList
            productionSlug={productionSlug}
            previewSlug={previewSlug}
            merged={merged}
            onToggleEnv={actions.toggleEnv}
            onAddToEnv={actions.addToEnv}
            onReorder={actions.reorder}
            onDelete={actions.delete}
            onEdit={panels.openEdit}
          />
        )}
        <PolicyPanel
          mode="add"
          productionSlug={productionSlug}
          previewSlug={previewSlug}
          isOpen={panels.isAddPanelOpen}
          onClose={panels.closeAdd}
          existingMatchKeys={existingMatchKeys}
          onSave={actions.save}
        />
        {editingPolicy !== null && (
          <PolicyPanel
            key={editingPolicy.id}
            mode="edit"
            productionSlug={productionSlug}
            previewSlug={previewSlug}
            isOpen={panels.isEditPanelOpen}
            onClose={panels.closeEdit}
            existingMatchKeys={existingMatchKeys}
            initialPolicy={editingPolicy}
            initialEnvironmentId={editingInitialEnvId}
            onSave={(prodPolicy, previewPolicy) => {
              actions.save(prodPolicy, previewPolicy, editingRow);
              panels.closeEdit();
            }}
          />
        )}
      </PageBody>
    </PageContainer>
  );
}

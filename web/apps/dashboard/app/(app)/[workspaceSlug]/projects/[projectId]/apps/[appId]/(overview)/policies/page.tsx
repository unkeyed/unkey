"use client";
import { TOP_NAV_HEIGHT } from "@/components/navigation/top-nav";
import { POLICY_LIMITS, policyIdentity } from "@/lib/collections/deploy/policies.schema";
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

  // By id, not by merge key: a duplicate name keys its row from one
  // environment's id, so a click on the other copy would not match.
  const editingRow = panels.editing
    ? merged.find(
        (m) => m.production?.id === panels.editing?.id || m.preview?.id === panels.editing?.id,
      )
    : undefined;
  const editingEnabled = {
    a: editingRow?.production?.enabled ?? false,
    b: editingRow?.preview?.enabled ?? false,
  };

  const existingIdentities = merged.map((m) => policyIdentity(m.type, m.name));

  const editingInitialEnvId =
    editingEnabled.a && editingEnabled.b
      ? "__all__"
      : editingEnabled.a
        ? productionSlug
        : editingEnabled.b
          ? previewSlug
          : "__all__";

  // An API write can leave the two copies apart, and the body on screen has to
  // be the body that gets written.
  const editingPolicy =
    editingInitialEnvId === previewSlug
      ? (editingRow?.preview ?? panels.editing)
      : (editingRow?.production ?? panels.editing);

  // An environment the row already exists in takes an update, not an insert.
  const isFull = {
    production: rowsByEnv.production.length >= POLICY_LIMITS.maxPolicies,
    preview: rowsByEnv.preview.length >= POLICY_LIMITS.maxPolicies,
  };
  const atCapacity = {
    add: isFull,
    edit: {
      production: editingRow?.production == null && isFull.production,
      preview: editingRow?.preview == null && isFull.preview,
    },
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
            <Plus iconSize="sm-regular" />
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
          topOffset={TOP_NAV_HEIGHT}
          onClose={panels.closeAdd}
          existingIdentities={existingIdentities}
          atCapacity={atCapacity.add}
          onSave={actions.save}
        />
        {editingPolicy !== null && (
          <PolicyPanel
            key={editingPolicy.id}
            mode="edit"
            productionSlug={productionSlug}
            previewSlug={previewSlug}
            isOpen={panels.isEditPanelOpen}
            topOffset={TOP_NAV_HEIGHT}
            onClose={panels.closeEdit}
            existingIdentities={existingIdentities}
            atCapacity={atCapacity.edit}
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

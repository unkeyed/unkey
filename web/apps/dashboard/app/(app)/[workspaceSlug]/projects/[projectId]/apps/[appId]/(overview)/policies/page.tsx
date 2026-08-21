"use client";
import { TOP_NAV_HEIGHT } from "@/components/navigation/top-nav";
import { POLICY_LIMITS } from "@/lib/collections/deploy/policies.schema";
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
  const { envAId, envBId, envASlug, envBSlug, merged, isLoading, isError } = usePoliciesData();
  const actions = usePolicyActions({ envAId, envBId, projectId, appId, merged });
  const panels = usePolicyPanels();

  // `panels.editing` is the copy the user clicked, from one environment. Find
  // the row holding it by id, not by merge key: a duplicate name keys its row
  // from one environment's id, so a click on the other copy would not match.
  const editingRow = panels.editing
    ? merged.find((m) => m.envA?.id === panels.editing?.id || m.envB?.id === panels.editing?.id)
    : undefined;
  const editingEnabled = {
    a: editingRow?.envA?.enabled ?? false,
    b: editingRow?.envB?.enabled ?? false,
  };

  const existingNames = merged.map((m) => m.name);

  const editingInitialEnvId =
    editingEnabled.a && editingEnabled.b
      ? "__all__"
      : editingEnabled.a
        ? envASlug
        : editingEnabled.b
          ? envBSlug
          : "__all__";

  // An API write can leave the two copies apart, and the body on screen has to
  // be the body that gets written.
  const editingPolicy =
    editingInitialEnvId === envBSlug
      ? (editingRow?.envB ?? panels.editing)
      : (editingRow?.envA ?? panels.editing);

  // An environment the row already exists in takes an update, not an insert.
  const atCapacity = {
    add: {
      a: merged.filter((m) => m.envA !== null).length >= POLICY_LIMITS.maxPolicies,
      b: merged.filter((m) => m.envB !== null).length >= POLICY_LIMITS.maxPolicies,
    },
    edit: {
      a:
        editingRow?.envA == null &&
        merged.filter((m) => m.envA !== null).length >= POLICY_LIMITS.maxPolicies,
      b:
        editingRow?.envB == null &&
        merged.filter((m) => m.envB !== null).length >= POLICY_LIMITS.maxPolicies,
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
        <PolicyPanel
          mode="add"
          envASlug={envASlug}
          envBSlug={envBSlug}
          isOpen={panels.isAddPanelOpen}
          topOffset={TOP_NAV_HEIGHT}
          onClose={panels.closeAdd}
          existingNames={existingNames}
          atCapacity={atCapacity.add}
          onSave={actions.save}
        />
        {editingPolicy !== null && (
          <PolicyPanel
            key={editingPolicy.id}
            mode="edit"
            envASlug={envASlug}
            envBSlug={envBSlug}
            isOpen={panels.isEditPanelOpen}
            topOffset={TOP_NAV_HEIGHT}
            onClose={panels.closeEdit}
            existingNames={existingNames}
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

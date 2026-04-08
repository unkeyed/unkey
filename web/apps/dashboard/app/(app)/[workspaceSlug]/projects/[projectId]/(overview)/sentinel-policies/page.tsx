"use client";

import { ProjectContentWrapper } from "../../components/project-content-wrapper";
import { useOptionalProjectLayout } from "../layout-provider";
import { SentinelPolicyPanel } from "./components/add-panel";
import { SentinelPoliciesList } from "./components/list";
import { SentinelPoliciesEmpty } from "./components/list/empty";
import { SentinelPoliciesHeader } from "./components/list/header";
import { SentinelPoliciesListSkeleton } from "./components/list/skeleton";
import { useSentinelPoliciesData } from "./hooks/use-sentinel-policies-data";
import { useSentinelPolicyActions } from "./hooks/use-sentinel-policy-actions";
import { useSentinelPolicyPanels } from "./hooks/use-sentinel-policy-panels";

export default function SentinelPoliciesPage() {
  const layout = useOptionalProjectLayout();
  const { envAId, envBId, envASlug, envBSlug, merged, isLoading } = useSentinelPoliciesData();
  const actions = useSentinelPolicyActions({ envAId, envBId });
  const panels = useSentinelPolicyPanels();

  return (
    <ProjectContentWrapper centered maxWidth="960px" className="mt-8">
      <div className="flex flex-col gap-5">
        <SentinelPoliciesHeader onAddPolicy={panels.openAdd} />
        {isLoading ? (
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
          topOffset={layout?.tableDistanceToTop ?? 0}
          onClose={panels.closeAdd}
          onAdd={actions.add}
        />
        {panels.editing !== null && (
          <SentinelPolicyPanel
            key={panels.editing.id}
            mode="edit"
            envASlug={envASlug}
            envBSlug={envBSlug}
            isOpen
            topOffset={layout?.tableDistanceToTop ?? 0}
            onClose={panels.closeEdit}
            initialPolicy={panels.editing}
            onSave={(updated) => {
              actions.save(updated);
              panels.closeEdit();
            }}
          />
        )}
      </div>
    </ProjectContentWrapper>
  );
}

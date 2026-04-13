"use client";
import { useState } from "react";
import { ProjectContentWrapper } from "../../components/project-content-wrapper";
import { useOptionalProjectLayout } from "../layout-provider";
import { SentinelPolicyPanel } from "./components/add-panel";
import { AiPolicyPrompt } from "./components/add-panel/ai-prompt";
import type { PolicyFormValues } from "./components/add-panel/schema";
import { SentinelPoliciesList } from "./components/list";
import { SentinelPoliciesEmpty } from "./components/list/empty";
import { SentinelPoliciesError } from "./components/list/error";
import { SentinelPoliciesHeader } from "./components/list/header";
import { SentinelPoliciesListSkeleton } from "./components/list/skeleton";
import { useSentinelPoliciesData } from "./hooks/use-sentinel-policies-data";
import { useSentinelPolicyActions } from "./hooks/use-sentinel-policy-actions";
import { useSentinelPolicyPanels } from "./hooks/use-sentinel-policy-panels";

export default function SentinelPoliciesPage() {
  const layout = useOptionalProjectLayout();
  const { envAId, envBId, envASlug, envBSlug, merged, isLoading, isError } =
    useSentinelPoliciesData();
  const actions = useSentinelPolicyActions({ envAId, envBId });
  const panels = useSentinelPolicyPanels();
  const [isAiPanelOpen, setIsAiPanelOpen] = useState(false);
  const [aiPreview, setAiPreview] = useState<PolicyFormValues[]>([]);

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
    <ProjectContentWrapper centered maxWidth="960px" className="mt-8">
      <div className="flex flex-col gap-5">
        <SentinelPoliciesHeader
          onAddPolicy={panels.openAdd}
          onGenerateWithAi={() => setIsAiPanelOpen(true)}
        />
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
        <AiPolicyPrompt
          isOpen={isAiPanelOpen}
          topOffset={layout?.tableDistanceToTop ?? 0}
          onClose={() => setIsAiPanelOpen(false)}
          preview={aiPreview}
          onPreviewChange={setAiPreview}
          onOpenAddPanel={(values, index) => {
            setIsAiPanelOpen(false);
            panels.openAdd(values, true, index);
          }}
        />
        <SentinelPolicyPanel
          key={panels.addKey}
          mode="add"
          initialValues={panels.addInitialValues ?? undefined}
          envASlug={envASlug}
          envBSlug={envBSlug}
          isOpen={panels.isAddPanelOpen}
          topOffset={layout?.tableDistanceToTop ?? 0}
          onClose={() => {
            panels.closeAdd();
            if (panels.addOpenedFromAi) {
              setIsAiPanelOpen(true);
            }
          }}
          onSave={(prodPolicy, previewPolicy) => {
            actions.save(prodPolicy, previewPolicy);
            if (panels.addOpenedFromAi && panels.addAiPreviewIndex !== null) {
              const idx = panels.addAiPreviewIndex;
              setAiPreview((prev) => prev.filter((_, i) => i !== idx));
            }
          }}
        />
        {panels.editing !== null && (
          <SentinelPolicyPanel
            key={panels.editing.id}
            mode="edit"
            envASlug={envASlug}
            envBSlug={envBSlug}
            isOpen={panels.isEditPanelOpen}
            topOffset={layout?.tableDistanceToTop ?? 0}
            onClose={panels.closeEdit}
            initialPolicy={panels.editing}
            initialEnvironmentId={editingInitialEnvId}
            onSave={(prodPolicy, previewPolicy) => {
              actions.save(prodPolicy, previewPolicy);
              panels.closeEdit();
            }}
          />
        )}
      </div>
    </ProjectContentWrapper>
  );
}

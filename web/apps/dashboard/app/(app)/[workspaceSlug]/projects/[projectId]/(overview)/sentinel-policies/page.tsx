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
  const actions = useSentinelPolicyActions({ envAId, envBId, envASlug, envBSlug });
  const panels = useSentinelPolicyPanels();
  const [isAiPanelOpen, setIsAiPanelOpen] = useState(false);
  const [aiPreview, setAiPreview] = useState<PolicyFormValues[]>([]);

  const { openAdd, closeAdd, addOpenedFromAi, addAiPreviewIndex } = panels;
  const { saveFromForm, save } = actions;

  const openAiPanel = () => setIsAiPanelOpen(true);
  const closeAiPanel = () => setIsAiPanelOpen(false);
  const openAddPolicy = () => openAdd();

  const handleOpenAddFromAi = (values: PolicyFormValues, index: number) => {
    setIsAiPanelOpen(false);
    openAdd(values, true, index);
  };

  const handleAddAll = () => {
    const keyauthPolicies: PolicyFormValues[] = [];
    const directSave: PolicyFormValues[] = [];

    for (const p of aiPreview) {
      if (p.type === "keyauth" && p.keySpaceIds.length === 0) {
        keyauthPolicies.push(p);
      } else {
        directSave.push(p);
      }
    }

    if (directSave.length > 0) {
      saveFromForm(directSave);
    }

    // Keyauth without a keyspace can't be saved; keep them in the preview so
    // the user can open each one and pick a keyspace before retrying.
    if (keyauthPolicies.length > 0) {
      setAiPreview(keyauthPolicies);
    } else {
      setAiPreview([]);
      setIsAiPanelOpen(false);
    }
  };

  const handleAddPanelDismiss = (values: PolicyFormValues) => {
    if (addOpenedFromAi && addAiPreviewIndex !== null) {
      const idx = addAiPreviewIndex;
      setAiPreview((prev) => prev.map((p, i) => (i === idx ? values : p)));
      setIsAiPanelOpen(true);
    }
  };

  const handleAddPanelQuit = (values: PolicyFormValues) => {
    if (addOpenedFromAi && addAiPreviewIndex !== null) {
      const idx = addAiPreviewIndex;
      setAiPreview((prev) => prev.map((p, i) => (i === idx ? values : p)));
    }
  };

  const handleAddPanelSave = (
    prodPolicy: Parameters<typeof save>[0],
    previewPolicy: Parameters<typeof save>[1],
  ) => {
    save(prodPolicy, previewPolicy);
    if (addOpenedFromAi && addAiPreviewIndex !== null) {
      const idx = addAiPreviewIndex;
      // Reopen the AI panel so the user can keep editing the remaining
      // preview rows; skip it once this was the last one.
      setAiPreview((prev) => {
        const next = prev.filter((_, i) => i !== idx);
        if (next.length > 0) {
          setIsAiPanelOpen(true);
        }
        return next;
      });
    }
  };

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
        <SentinelPoliciesHeader onAddPolicy={openAddPolicy} onGenerateWithAi={openAiPanel} />
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
          onClose={closeAiPanel}
          preview={aiPreview}
          onPreviewChange={setAiPreview}
          onOpenAddPanel={handleOpenAddFromAi}
          onAddAll={handleAddAll}
        />
        <SentinelPolicyPanel
          mode="add"
          initialValues={panels.addInitialValues ?? undefined}
          openedFromAi={addOpenedFromAi}
          envASlug={envASlug}
          envBSlug={envBSlug}
          isOpen={panels.isAddPanelOpen}
          topOffset={layout?.tableDistanceToTop ?? 0}
          onClose={closeAdd}
          onDismiss={handleAddPanelDismiss}
          onQuit={handleAddPanelQuit}
          onSave={handleAddPanelSave}
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

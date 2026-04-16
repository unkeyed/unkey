"use client";

import type { SentinelPolicy } from "@/lib/collections/deploy/sentinel-policies.schema";
import { useCallback, useState } from "react";
import type { PolicyFormValues } from "../components/add-panel/schema";

export function useSentinelPolicyPanels() {
  const [isAddPanelOpen, setIsAddPanelOpen] = useState(false);
  const [addInitialValues, setAddInitialValues] = useState<PolicyFormValues | null>(null);
  const [addKey, setAddKey] = useState(0);
  const [addOpenedFromAi, setAddOpenedFromAi] = useState(false);
  const [addAiPreviewIndex, setAddAiPreviewIndex] = useState<number | null>(null);
  const [isEditPanelOpen, setIsEditPanelOpen] = useState(false);
  const [editing, setEditing] = useState<SentinelPolicy | null>(null);

  const openAdd = useCallback(
    (initialValues?: PolicyFormValues, fromAi = false, aiPreviewIndex?: number) => {
      setAddInitialValues(initialValues ?? null);
      setAddOpenedFromAi(fromAi);
      setAddAiPreviewIndex(aiPreviewIndex ?? null);
      setAddKey((k) => k + 1);
      setIsAddPanelOpen(true);
    },
    [],
  );

  const closeAdd = useCallback(() => {
    setIsAddPanelOpen(false);
    setAddInitialValues(null);
  }, []);

  const openEdit = useCallback((policy: SentinelPolicy) => {
    setEditing(policy);
    // Delay open by a frame so panel mounts first, then animates in
    requestAnimationFrame(() => {
      setIsEditPanelOpen(true);
    });
  }, []);
  const closeEdit = useCallback(() => setIsEditPanelOpen(false), []);

  return {
    isAddPanelOpen,
    addInitialValues,
    addKey,
    addOpenedFromAi,
    addAiPreviewIndex,
    openAdd,
    closeAdd,
    editing,
    isEditPanelOpen,
    openEdit,
    closeEdit,
  };
}

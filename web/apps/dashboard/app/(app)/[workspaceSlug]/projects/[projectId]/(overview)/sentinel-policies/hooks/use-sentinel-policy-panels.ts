"use client";

import type { SentinelPolicy } from "@/lib/collections/deploy/sentinel-policies.schema";
import { useCallback, useState } from "react";

export function useSentinelPolicyPanels() {
  const [isAddPanelOpen, setIsAddPanelOpen] = useState(false);
  const [isEditPanelOpen, setIsEditPanelOpen] = useState(false);
  const [editing, setEditing] = useState<SentinelPolicy | null>(null);

  const openAdd = useCallback(() => setIsAddPanelOpen(true), []);
  const closeAdd = useCallback(() => setIsAddPanelOpen(false), []);
  const openEdit = useCallback((policy: SentinelPolicy) => {
    setEditing(policy);
    // Delay open by a frame so panel mounts first, then animates in
    requestAnimationFrame(() => {
      setIsEditPanelOpen(true);
    });
  }, []);
  const closeEdit = useCallback(() => setIsEditPanelOpen(false), []);

  return { isAddPanelOpen, openAdd, closeAdd, editing, isEditPanelOpen, openEdit, closeEdit };
}

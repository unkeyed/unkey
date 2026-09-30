"use client";

import type { Policy } from "@/lib/collections/deploy/policies.schema";
import { useCallback, useState } from "react";

export function usePolicyPanels() {
  const [isAddPanelOpen, setIsAddPanelOpen] = useState(false);
  const [isEditPanelOpen, setIsEditPanelOpen] = useState(false);
  const [editing, setEditing] = useState<Policy | null>(null);

  const openAdd = useCallback(() => setIsAddPanelOpen(true), []);
  const closeAdd = useCallback(() => setIsAddPanelOpen(false), []);
  const openEdit = useCallback((policy: Policy) => {
    setEditing(policy);
    // Delay open by a frame so panel mounts first, then animates in
    requestAnimationFrame(() => {
      setIsEditPanelOpen(true);
    });
  }, []);
  const closeEdit = useCallback(() => setIsEditPanelOpen(false), []);

  return { isAddPanelOpen, openAdd, closeAdd, editing, isEditPanelOpen, openEdit, closeEdit };
}

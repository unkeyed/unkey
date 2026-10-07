"use client";

import { useCallback, useState } from "react";

export function usePolicyPanels() {
  const [isAddPanelOpen, setIsAddPanelOpen] = useState(false);
  const [addSession, setAddSession] = useState(0);
  const [isEditPanelOpen, setIsEditPanelOpen] = useState(false);
  const [editing, setEditing] = useState<{ key: string; session: number } | null>(null);
  const [isGuideOpen, setIsGuideOpen] = useState(false);

  const openAdd = useCallback(() => {
    setAddSession((prev) => prev + 1);
    requestAnimationFrame(() => {
      setIsAddPanelOpen(true);
    });
  }, []);
  const closeAdd = useCallback(() => setIsAddPanelOpen(false), []);
  const openEdit = useCallback((key: string) => {
    setEditing((prev) => ({ key, session: (prev?.session ?? 0) + 1 }));
    // Delay open by a frame so panel mounts first, then animates in
    requestAnimationFrame(() => {
      setIsEditPanelOpen(true);
    });
  }, []);
  const closeEdit = useCallback(() => setIsEditPanelOpen(false), []);
  const openGuide = useCallback(() => setIsGuideOpen(true), []);
  const closeGuide = useCallback(() => setIsGuideOpen(false), []);

  return {
    isAddPanelOpen,
    addSession,
    openAdd,
    closeAdd,
    editing,
    isEditPanelOpen,
    openEdit,
    closeEdit,
    isGuideOpen,
    openGuide,
    closeGuide,
  };
}

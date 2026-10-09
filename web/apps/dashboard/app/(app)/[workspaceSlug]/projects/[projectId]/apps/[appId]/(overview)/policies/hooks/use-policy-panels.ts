"use client";

import { useCallback, useState } from "react";
import type { PolicyRowKey } from "../components/list/merge";

export type PolicyPanel = { type: "add" } | { type: "edit"; key: PolicyRowKey } | { type: "guide" };

type PanelState = { panel: PolicyPanel; session: number; isOpen: boolean };

export function usePolicyPanels() {
  const [state, setState] = useState<PanelState | null>(null);

  const open = useCallback((panel: PolicyPanel) => {
    setState((prev) => ({ panel, session: (prev?.session ?? 0) + 1, isOpen: false }));
    // Base UI only animates an open transition, and `session` remounts the panel.
    requestAnimationFrame(() => setState((prev) => prev && { ...prev, isOpen: true }));
  }, []);
  const close = useCallback(() => setState((prev) => prev && { ...prev, isOpen: false }), []);

  return {
    panel: state?.panel ?? null,
    session: state?.session ?? 0,
    isOpen: state?.isOpen ?? false,
    open,
    close,
  };
}

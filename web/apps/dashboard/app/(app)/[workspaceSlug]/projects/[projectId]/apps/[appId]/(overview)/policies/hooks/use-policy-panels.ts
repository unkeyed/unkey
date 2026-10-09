"use client";

import { useCallback, useState } from "react";

export type PolicyPanel = { type: "add" } | { type: "edit"; key: string } | { type: "guide" };

type PanelState = { panel: PolicyPanel; session: number; isOpen: boolean };

/**
 * One panel at a time. A closed panel stays mounted so it can animate out.
 * `session` changes on every open, so each open starts from a fresh form.
 */
export function usePolicyPanels() {
  const [state, setState] = useState<PanelState | null>(null);

  const open = useCallback((panel: PolicyPanel) => {
    setState((prev) => ({ panel, session: (prev?.session ?? 0) + 1, isOpen: false }));
    // Mount closed first, so the panel animates in on the next frame.
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

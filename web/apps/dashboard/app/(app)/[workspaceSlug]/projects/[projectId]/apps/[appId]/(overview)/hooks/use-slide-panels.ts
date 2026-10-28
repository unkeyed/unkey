"use client";

import { useCallback, useState } from "react";

type PanelState<P> = { panel: P; session: number; isOpen: boolean };

export function useSlidePanels<P>() {
  const [state, setState] = useState<PanelState<P> | null>(null);

  const open = useCallback((panel: P) => {
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

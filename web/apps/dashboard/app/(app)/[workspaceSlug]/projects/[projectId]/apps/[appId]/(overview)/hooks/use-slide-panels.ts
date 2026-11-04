"use client";

import { useCallback, useEffect, useRef, useState } from "react";

type PanelState<P> = { panel: P; session: number; isOpen: boolean };

export function useSlidePanels<P>() {
  const [state, setState] = useState<PanelState<P> | null>(null);
  const frame = useRef<number | null>(null);
  const latest = useRef(0);
  const session = state?.session ?? 0;

  useEffect(
    () => () => {
      if (frame.current !== null) {
        cancelAnimationFrame(frame.current);
      }
    },
    [],
  );

  const open = useCallback((panel: P) => {
    if (frame.current !== null) {
      cancelAnimationFrame(frame.current);
    }
    const next = ++latest.current;
    setState({ panel, session: next, isOpen: false });
    // Base UI only animates an open transition, and `session` remounts the panel.
    frame.current = requestAnimationFrame(() => {
      frame.current = null;
      setState((prev) => (prev?.session === next ? { ...prev, isOpen: true } : prev));
    });
  }, []);
  const close = useCallback(() => {
    if (frame.current !== null && latest.current === session) {
      cancelAnimationFrame(frame.current);
      frame.current = null;
    }
    setState((prev) => (prev && prev.session === session ? { ...prev, isOpen: false } : prev));
  }, [session]);

  return {
    panel: state?.panel ?? null,
    session,
    isOpen: state?.isOpen ?? false,
    open,
    close,
  };
}

"use client";

import { type ReactNode, useEffect, useRef, useState } from "react";

type PresenceState = {
  closing: boolean;
  presentOnMount: boolean;
};

/**
 * Keeps children mounted for `exitMs` after `present` turns false so CSS can animate them out.
 * While closing, the children from the last present render are shown, so content does not
 * change under the exit animation.
 */
export function Presence({
  present,
  exitMs,
  children,
}: {
  present: boolean;
  exitMs: number;
  children: (state: PresenceState) => ReactNode;
}) {
  const [mounted, setMounted] = useState(present);
  const presentOnMount = useRef(present);
  const lastChildren = useRef(children);
  if (present) {
    lastChildren.current = children;
  }

  useEffect(() => {
    if (present) {
      setMounted(true);
      return;
    }
    presentOnMount.current = false;
    const timer = setTimeout(() => setMounted(false), exitMs);
    return () => clearTimeout(timer);
  }, [present, exitMs]);

  if (!present && !mounted) {
    return null;
  }
  return lastChildren.current({ closing: !present, presentOnMount: presentOnMount.current });
}

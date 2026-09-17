"use client";

import { useEffect, useState } from "react";

// Relative labels computed with Date.now() during render freeze at mount and
// then jump whenever an unrelated re-render recomputes them.
export function useNow(intervalMs: number): number {
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    const interval = setInterval(() => setNow(Date.now()), intervalMs);
    return () => clearInterval(interval);
  }, [intervalMs]);

  return now;
}

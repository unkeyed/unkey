"use client";

import { type ReactNode, createContext, use, useState } from "react";

const HoveredBucket = createContext<number | null>(null);
const SetHoveredBucket = createContext<(ms: number | null) => void>(() => {});

// Split from the setter so a scrub re-renders only the legend lines, not every chart.
export function ChartHoverProvider({ children }: { children: ReactNode }) {
  const [hoveredMs, setHoveredMs] = useState<number | null>(null);
  return (
    <SetHoveredBucket value={setHoveredMs}>
      <HoveredBucket value={hoveredMs}>{children}</HoveredBucket>
    </SetHoveredBucket>
  );
}

export function useHoveredBucket(): number | null {
  return use(HoveredBucket);
}

export function useSetHoveredBucket(): (ms: number | null) => void {
  return use(SetHoveredBucket);
}

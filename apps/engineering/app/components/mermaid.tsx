"use client";

import mermaid from "mermaid";
import { useTheme } from "next-themes";
import { useEffect, useRef } from "react";

export function Mermaid({ chart }: { chart: string }) {
  const ref = useRef<HTMLDivElement>(null);
  const { resolvedTheme } = useTheme();

  useEffect(() => {
    if (!ref.current) {
      return;
    }
    mermaid.initialize({
      startOnLoad: false,
      theme: resolvedTheme === "dark" ? "dark" : "default",
    });

    void mermaid.run({
      nodes: [ref.current],
    });
  }, [resolvedTheme]);

  return (
    <div className="my-4">
      <div ref={ref} className="mermaid">
        {chart}
      </div>
    </div>
  );
}

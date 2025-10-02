'use client';

import { useTheme } from 'next-themes';
import { useEffect, useId, useRef } from 'react';
import mermaid from 'mermaid';

export function Mermaid({ chart }: { chart: string }) {
  const id = useId();
  const ref = useRef<HTMLDivElement>(null);
  const { resolvedTheme } = useTheme();

  useEffect(() => {
    if (!ref.current) return;

    mermaid.initialize({
      startOnLoad: false,
      theme: resolvedTheme === 'dark' ? 'dark' : 'default',
    });

    void mermaid.run({
      nodes: [ref.current],
    });
  }, [chart, resolvedTheme]);

  return (
    <div className="my-4">
      <div ref={ref} className="mermaid">
        {chart}
      </div>
    </div>
  );
}

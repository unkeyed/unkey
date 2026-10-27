"use client";

import { cn } from "@unkey/ui";
import { useId, useLayoutEffect, useState } from "react";

type Wire = { id: string; d: string };

export function ConnectionWires({
  root,
  highlightedId,
}: {
  root: HTMLDivElement | null;
  highlightedId: string | null;
}) {
  const [wires, setWires] = useState<Wire[]>([]);
  const markerId = useId();

  useLayoutEffect(() => {
    if (!root) {
      setWires([]);
      return;
    }
    const measure = () => {
      const from = root.querySelector<HTMLElement>("[data-wire-from]");
      if (!from) {
        setWires([]);
        return;
      }
      const base = root.getBoundingClientRect();
      const source = from.getBoundingClientRect();
      const path = (target: DOMRect) => {
        const x2 = target.left - base.left - 6;
        const y2 = target.top - base.top + target.height / 2;
        if (source.right > target.left) {
          const x1 = source.left - base.left + 12;
          const y1 = source.bottom - base.top;
          return `M ${x1} ${y1} V ${y2 - 8} Q ${x1} ${y2} ${x1 + 8} ${y2} H ${x2}`;
        }
        const x1 = source.right - base.left;
        const y1 = source.top - base.top + source.height / 2;
        const mid = Math.round((x1 + x2) / 2);
        const radius = Math.min(10, Math.abs(y2 - y1) / 2);
        const direction = Math.sign(y2 - y1);
        return `M ${x1} ${y1} H ${mid - radius} Q ${mid} ${y1} ${mid} ${y1 + direction * radius} V ${y2 - direction * radius} Q ${mid} ${y2} ${mid + radius} ${y2} H ${x2}`;
      };
      const next: Wire[] = [];
      for (const element of root.querySelectorAll<HTMLElement>("[data-wire-to]")) {
        const id = element.dataset.wireTo;
        if (id) {
          next.push({ id, d: path(element.getBoundingClientRect()) });
        }
      }
      setWires(next);
    };
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(root);
    for (const child of root.querySelectorAll("[data-wire-to], [data-wire-from]")) {
      observer.observe(child);
    }
    return () => observer.disconnect();
  }, [root]);

  return (
    <svg
      aria-hidden="true"
      className="pointer-events-none absolute inset-0 z-10 h-full w-full overflow-visible"
    >
      <defs>
        <marker
          id={`${markerId}-source`}
          viewBox="0 0 8 8"
          refX="4"
          refY="4"
          markerWidth="8"
          markerHeight="8"
        >
          <circle cx="4" cy="4" r="2.5" className="fill-gray-2 stroke-gray-8" />
        </marker>
        <marker
          id={markerId}
          viewBox="0 0 6 6"
          refX="5"
          refY="3"
          markerWidth="6"
          markerHeight="6"
          orient="auto-start-reverse"
        >
          <path d="M 0 0 L 5 3 L 0 6" className="stroke-gray-9" fill="none" />
        </marker>
      </defs>
      <g
        fill="none"
        strokeLinejoin="round"
        strokeLinecap="round"
        strokeWidth={1}
        strokeDasharray="3 3"
      >
        {wires.map((wire) => (
          <path
            key={wire.id}
            d={wire.d}
            markerStart={`url(#${markerId}-source)`}
            markerEnd={`url(#${markerId})`}
            data-connection-edge={wire.id}
            className={cn(
              "stroke-gray-7",
              highlightedId === wire.id &&
                "animate-dash-flow stroke-gray-11 [animation-duration:1.6s] motion-reduce:animate-none",
            )}
          />
        ))}
      </g>
    </svg>
  );
}

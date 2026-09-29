"use client";

import { IconChevronDownOutline12 } from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import { type ReactNode, useRef, useState } from "react";
import { usePan } from "./use-pan";

const COLLAPSED_HEIGHT_PX = 360;

export function CanvasViewport({
  label,
  className,
  children,
}: {
  label: string;
  className?: string;
  children: ReactNode;
}) {
  const viewportRef = useRef<HTMLDivElement>(null);
  const layerRef = useRef<HTMLDivElement>(null);
  const [expanded, setExpanded] = useState(false);
  const { offset, phase, pannable, contentHeight, viewportProps } = usePan(viewportRef, layerRef);
  const overflows = contentHeight > COLLAPSED_HEIGHT_PX;

  return (
    <div className="relative">
      <section
        ref={viewportRef}
        // biome-ignore lint/a11y/noNoninteractiveTabindex: focusable so arrow keys can pan the canvas
        tabIndex={0}
        aria-label={label}
        className={cn(
          "touch-pan-y select-none overflow-hidden outline-none transition-[max-height] duration-200 ease-out focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-gray-7 motion-reduce:transition-none",
          phase === "panning" ? "cursor-grabbing [&_*]:cursor-grabbing" : pannable && "cursor-grab",
        )}
        style={{ maxHeight: expanded ? contentHeight : COLLAPSED_HEIGHT_PX }}
        {...viewportProps}
      >
        <div
          ref={layerRef}
          className={cn("relative", className)}
          style={{ transform: `translate3d(${offset.x}px, ${offset.y}px, 0)` }}
        >
          {children}
        </div>
      </section>
      {overflows && (
        <button
          type="button"
          onClick={() => setExpanded((e) => !e)}
          aria-expanded={expanded}
          aria-label={expanded ? "Collapse canvas" : "Expand canvas"}
          className="absolute bottom-2 left-1/2 z-10 flex size-6 -translate-x-1/2 items-center justify-center rounded-full border bg-raised text-gray-11 shadow-xs transition-colors hover:border-gray-10 hover:text-gray-12 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-7"
        >
          <IconChevronDownOutline12
            className={cn(
              "size-3 transition-transform duration-200 ease-out motion-reduce:transition-none",
              expanded && "rotate-180",
            )}
          />
        </button>
      )}
    </div>
  );
}

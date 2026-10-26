"use client";

import { Button, HoverCard, HoverCardContent, HoverCardTrigger } from "@unkey/ui";
import { type ReactNode, useState } from "react";
import { ReferenceArea, ReferenceLine } from "recharts";

export type ChartMarker = { id: string; atMs: number; span: { endMs: number | null } | null };

export type ChartMarkers = {
  items: ChartMarker[];
  activeId: string | null;
  onActiveChange: (id: string | null) => void;
  renderPopover: (id: string) => ReactNode;
};

type Domain = [number, number];

type PlacedMarker = { id: string; atMs: number; leftPct: number };

export function placeMarkers(items: ChartMarker[], [start, end]: Domain): PlacedMarker[] {
  const width = end - start;
  if (width <= 0) {
    return [];
  }
  return items
    .filter((m) => m.atMs >= start && m.atMs <= end)
    .map((m) => ({ id: m.id, atMs: m.atMs, leftPct: ((m.atMs - start) / width) * 100 }));
}

export function activeMarkerSpan(
  items: ChartMarker[],
  activeId: string | null,
  [start, end]: Domain,
): Domain | null {
  const active = activeId ? items.find((m) => m.id === activeId) : undefined;
  if (!active?.span) {
    return null;
  }
  const from = Math.max(active.atMs, start);
  const to = Math.min(active.span.endMs ?? end, end);
  return from < to ? [from, to] : null;
}

export function ChartMarkerOverlay({
  markers,
  domain,
  insetLeft,
  insetRight,
  bandHeight,
}: {
  markers: ChartMarkers;
  domain: Domain;
  insetLeft: number;
  insetRight: number;
  bandHeight: number;
}) {
  const [ownerId, setOwnerId] = useState<string | null>(null);
  const { activeId, onActiveChange, renderPopover } = markers;

  return (
    <div
      className="pointer-events-none absolute top-0"
      style={{ left: insetLeft, right: insetRight, height: bandHeight }}
    >
      {placeMarkers(markers.items, domain).map((m) => {
        const isActive = activeId === m.id;
        return (
          <HoverCard
            key={m.id}
            open={isActive && ownerId === m.id}
            onOpenChange={(open) => {
              if (open) {
                setOwnerId(m.id);
                onActiveChange(m.id);
                return;
              }
              setOwnerId(null);
              if (isActive) {
                onActiveChange(null);
              }
            }}
          >
            <HoverCardTrigger
              delay={0}
              closeDelay={100}
              render={<Button variant="ghost" size="icon" />}
              aria-label={`Marker at ${new Date(m.atMs).toLocaleString()}`}
              data-active={isActive || undefined}
              className="pointer-events-auto absolute bottom-0 -translate-x-1/2 translate-y-1/2 rounded-full p-0"
              style={{ left: `${m.leftPct}%` }}
            >
              <span className="size-2 rounded-full border border-gray-9 bg-background transition-[scale,background-color,border-color] duration-150 ease-out group-hover:scale-125 group-hover:border-gray-12 group-data-active:scale-125 group-data-active:border-gray-12 group-data-active:bg-gray-12 motion-reduce:transition-none" />
            </HoverCardTrigger>
            <HoverCardContent
              side="top"
              sideOffset={6}
              className="w-72 p-3 motion-reduce:transition-none"
            >
              {renderPopover(m.id)}
            </HoverCardContent>
          </HoverCard>
        );
      })}
    </div>
  );
}

export function MarkerLayer({ markers, span }: { markers: ChartMarkers; span: Domain | null }) {
  return (
    <>
      {span && (
        <ReferenceArea
          x1={span[0]}
          x2={span[1]}
          fill="var(--color-grayA-3)"
          fillOpacity={1}
          strokeOpacity={0}
          className="animate-in fade-in-0 duration-150 motion-reduce:animate-none"
        />
      )}
      {markers.items.map((m) => (
        <ReferenceLine
          key={m.id}
          x={m.atMs}
          stroke={m.id === markers.activeId ? "var(--color-gray-10)" : "var(--color-gray-8)"}
          strokeDasharray="2 3"
          ifOverflow="hidden"
        />
      ))}
    </>
  );
}

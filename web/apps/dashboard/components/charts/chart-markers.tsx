"use client";

import { plural } from "@/lib/fmt";
import { Button, HoverCard, HoverCardContent, HoverCardTrigger } from "@unkey/ui";
import { cn } from "cn";
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

type PlacedMarker = {
  key: string;
  ids: string[];
  atMs: number;
  x: number;
  leftPct: number;
  late: boolean;
};

// Markers past the domain end share one edge marker so they do not stack.
export function placeMarkers(items: ChartMarker[], [start, end]: Domain): PlacedMarker[] {
  const width = end - start;
  if (width <= 0) {
    return [];
  }
  const inside = items.filter((m) => m.atMs >= start && m.atMs <= end);
  const late = items.filter((m) => m.atMs > end);
  const placed = inside.map((m) => ({
    key: m.id,
    ids: [m.id],
    atMs: m.atMs,
    x: m.atMs,
    leftPct: ((m.atMs - start) / width) * 100,
    late: false,
  }));
  return late.length > 0
    ? [
        ...placed,
        {
          key: "late",
          ids: late.map((m) => m.id),
          atMs: Math.max(...late.map((m) => m.atMs)),
          x: end,
          leftPct: 100,
          late: true,
        },
      ]
    : placed;
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
        const isActive = activeId !== null && m.ids.includes(activeId);
        return (
          <HoverCard
            key={m.key}
            open={isActive && ownerId === m.key}
            onOpenChange={(open) => {
              if (open) {
                setOwnerId(m.key);
                onActiveChange(m.ids.at(-1) ?? null);
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
              aria-label={
                m.late
                  ? `${plural(m.ids.length, "marker")} after the last data point`
                  : `Marker at ${new Date(m.atMs).toLocaleString()}`
              }
              data-active={isActive || undefined}
              className="pointer-events-auto absolute bottom-0 -translate-x-1/2 translate-y-1/2 rounded-full p-0"
              style={{ left: `${m.leftPct}%` }}
            >
              <span
                className={cn(
                  "size-2 rounded-full border border-gray-9 bg-background transition-[scale,background-color,border-color] duration-150 ease-out group-hover:scale-125 group-hover:border-gray-12 group-data-active:scale-125 group-data-active:border-gray-12 group-data-active:bg-gray-12 motion-reduce:transition-none",
                  m.late && "border-dashed",
                )}
              />
            </HoverCardTrigger>
            <HoverCardContent
              side="top"
              sideOffset={6}
              className="w-72 p-3 motion-reduce:transition-none"
            >
              <div className="flex max-h-80 flex-col gap-3 divide-y divide-grayA-4 overflow-y-auto [&>*:not(:first-child)]:pt-3">
                {m.ids.map((id) => (
                  <div key={id}>{renderPopover(id)}</div>
                ))}
              </div>
            </HoverCardContent>
          </HoverCard>
        );
      })}
    </div>
  );
}

export function MarkerLayer({
  markers,
  domain,
  span,
}: {
  markers: ChartMarkers;
  domain: Domain;
  span: Domain | null;
}) {
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
      {placeMarkers(markers.items, domain).map((m) => (
        <ReferenceLine
          key={m.key}
          x={m.x}
          stroke={
            markers.activeId !== null && m.ids.includes(markers.activeId)
              ? "var(--color-gray-10)"
              : "var(--color-gray-8)"
          }
          strokeDasharray="2 3"
          ifOverflow="hidden"
        />
      ))}
    </>
  );
}

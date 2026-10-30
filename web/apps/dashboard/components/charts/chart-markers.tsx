"use client";

import { plural } from "@/lib/fmt";
import { IconCodeCommitOutline18 } from "@unkey/icons";
import { Button, HoverCard, HoverCardContent, HoverCardTrigger } from "@unkey/ui";
import type { ReactNode } from "react";
import { ReferenceArea, ReferenceLine } from "recharts";
import type { Domain } from "./chart-domain";
import { formatSpan } from "./format-interval";
import { type ActiveChange, type ActiveMarker, useHoverOwner } from "./use-hover-owner";

export type ChartMarker = {
  id: string;
  atMs: number;
  span: { endMs: number | null } | null;
};

export type ChartMarkers = {
  items: ChartMarker[];
  active: ActiveMarker;
  onActiveChange: ActiveChange;
  renderPopover: (id: string) => ReactNode;
};

type PlacedMarker = {
  key: string;
  ids: string[];
  fromMs: number;
  toMs: number;
  x: number;
  leftPct: number;
  title: string;
};

function groupTitle(count: number): string {
  return count === 1 ? "Deploy" : plural(count, "deploy");
}

// Markers past the domain end share one edge marker so they do not stack.
export function placeMarkers(items: ChartMarker[], [start, end]: Domain): PlacedMarker[] {
  const width = end - start;
  if (width <= 0) {
    return [];
  }
  const placed = items
    .filter((m) => m.atMs >= start && m.atMs <= end)
    .toSorted((a, b) => a.atMs - b.atMs)
    .map((m) => ({
      key: m.id,
      ids: [m.id],
      fromMs: m.atMs,
      toMs: m.atMs,
      x: m.atMs,
      leftPct: ((m.atMs - start) / width) * 100,
      title: groupTitle(1),
    }));
  const late = items.filter((m) => m.atMs > end).toSorted((a, b) => a.atMs - b.atMs);
  return late.length > 0
    ? [
        ...placed,
        {
          key: "late",
          ids: late.map((m) => m.id),
          fromMs: late[0].atMs,
          toMs: late[late.length - 1].atMs,
          x: end,
          leftPct: 100,
          title: groupTitle(late.length),
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

function markerSummary(m: PlacedMarker): string {
  return `${m.title} · ${formatSpan(m.fromMs, m.toMs, true)}`;
}

export function ChartMarkerOverlay({
  markers,
  placed,
  insetLeft,
  insetRight,
  bandHeight,
}: {
  markers: ChartMarkers;
  placed: PlacedMarker[];
  insetLeft: number;
  insetRight: number;
  bandHeight: number;
}) {
  const { active, onActiveChange, renderPopover } = markers;
  const hover = useHoverOwner(
    active,
    onActiveChange,
    placed.map((m) => m.key),
  );

  return (
    <div
      className="pointer-events-none absolute top-0"
      style={{ left: insetLeft, right: insetRight, height: bandHeight }}
    >
      {placed.map((m) => {
        const isActive = active !== null && m.ids.includes(active.id);
        const summary = markerSummary(m);
        return (
          <HoverCard key={m.key} {...hover(m.key, m.ids[m.ids.length - 1])}>
            <HoverCardTrigger
              delay={0}
              closeDelay={100}
              render={<Button variant="ghost" size="icon" />}
              aria-label={summary}
              data-active={isActive || undefined}
              className="pointer-events-auto absolute bottom-0 -translate-x-1/2 translate-y-1/2 rounded-full bg-transparent p-0 hover:bg-transparent focus:ring-0 focus-visible:ring-2 focus-visible:ring-gray-7"
              style={{ left: `${m.leftPct}%` }}
            >
              <IconCodeCommitOutline18
                aria-hidden="true"
                className="absolute inset-0 m-auto size-3.5 shrink-0 text-gray-9 transition-colors duration-150 ease-out group-hover:text-gray-12 group-data-active:text-gray-12 motion-reduce:transition-none [&_*]:stroke-1 [&_*]:[vector-effect:non-scaling-stroke] [&_circle]:fill-[var(--background-color-raised)]"
              />
            </HoverCardTrigger>
            <HoverCardContent
              side="bottom"
              sideOffset={6}
              collisionAvoidance={{ side: "none" }}
              className="w-72 p-3 motion-reduce:transition-none"
            >
              {m.ids.length > 1 && (
                <p className="mb-3 border-b border-grayA-4 pb-2 text-xs font-medium text-gray-12">
                  {summary}
                </p>
              )}
              <div className="flex max-h-80 flex-col divide-y divide-grayA-4 overflow-y-auto [&>*:not(:first-child)]:pt-3 [&>*:not(:last-child)]:pb-3">
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
  placed,
  span,
}: {
  markers: ChartMarkers;
  placed: PlacedMarker[];
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
      {placed.map((m) => (
        <ReferenceLine
          key={m.key}
          x={m.x}
          stroke={
            markers.active !== null && m.ids.includes(markers.active.id)
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

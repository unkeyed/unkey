"use client";

import { plural } from "@/lib/fmt";
import { Button, HoverCard, HoverCardContent, HoverCardTrigger } from "@unkey/ui";
import { cn } from "cn";
import { type CSSProperties, type ReactNode, useState } from "react";
import { ReferenceArea, ReferenceLine } from "recharts";

export type MarkerShape = "dot" | "diamond";

export type ChartMarker = {
  id: string;
  atMs: number;
  span: { endMs: number | null } | null;
  shape: MarkerShape;
  label: string | null;
};

const SHAPE_STYLE: Record<
  MarkerShape,
  { mark: string; band: string; line: string; activeLine: string }
> = {
  dot: {
    mark: "size-2 rounded-full border-gray-9 bg-background group-hover:border-gray-12 group-data-active:border-gray-12 group-data-active:bg-gray-12",
    band: "var(--color-grayA-3)",
    line: "var(--color-gray-8)",
    activeLine: "var(--color-gray-10)",
  },
  diamond: {
    mark: "size-1.5 rotate-45 rounded-[1px] border-error-9 bg-error-3 ring-2 ring-background group-hover:border-error-11 group-data-active:border-error-11 group-data-active:bg-error-9",
    band: "var(--color-errorA-3)",
    line: "var(--color-error-9)",
    activeLine: "var(--color-error-11)",
  },
};

const STACK_CAP = 4;

const STACK_STEP_PX = 10;

export function stackGapPct(plotWidthPx: number): number {
  return plotWidthPx > 0 ? (STACK_STEP_PX / plotWidthPx) * 100 : 0;
}

export type ChartMarkers = {
  items: ChartMarker[];
  activeId: string | null;
  onActiveChange: (id: string | null) => void;
  renderPopover: (id: string) => ReactNode;
};

type Domain = [number, number];

export type PlacedMarker = {
  key: string;
  ids: string[];
  atMs: number;
  x: number;
  leftPct: number;
  late: boolean;
  shape: MarkerShape;
  label: string | null;
};

// Markers past the domain end share one edge marker so they do not stack.
export function placeMarkers(
  items: ChartMarker[],
  [start, end]: Domain,
  stackGapPct: number,
): PlacedMarker[] {
  const width = end - start;
  if (width <= 0) {
    return [];
  }
  const inside = items
    .filter((m) => m.atMs >= start && m.atMs <= end)
    .toSorted((a, b) => a.atMs - b.atMs);
  const late = items.filter((m) => m.atMs > end);
  const placed: PlacedMarker[] = [];
  for (const m of inside) {
    const leftPct = ((m.atMs - start) / width) * 100;
    const previous = placed.at(-1);
    if (
      previous &&
      previous.shape === "diamond" &&
      m.shape === "diamond" &&
      leftPct - previous.leftPct < stackGapPct
    ) {
      previous.ids.push(m.id);
      previous.label = null;
      continue;
    }
    placed.push({
      key: m.id,
      ids: [m.id],
      atMs: m.atMs,
      x: m.atMs,
      leftPct,
      late: false,
      shape: m.shape,
      label: m.label,
    });
  }
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
          shape: late.every((m) => m.shape === "diamond") ? "diamond" : "dot",
          label: null,
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

type MarkerSlot = {
  key: string;
  ids: string[];
  layer: number;
  label: string;
  overflow: number;
};

export function markerSlots(
  m: PlacedMarker,
  labels: ReadonlyMap<string, string | null>,
): MarkerSlot[] {
  if (m.shape !== "diamond" || m.late || m.ids.length === 1) {
    return [{ key: m.key, ids: m.ids, layer: 0, label: markerLabel(m), overflow: 0 }];
  }
  const at = new Date(m.atMs).toLocaleString();
  const shown = m.ids.slice(0, STACK_CAP).map((id, layer) => ({
    key: id,
    ids: [id],
    layer,
    label: `${labels.get(id) ?? "Marker"} at ${at}`,
    overflow: 0,
  }));
  const rest = m.ids.slice(STACK_CAP);
  return rest.length === 0
    ? shown
    : [
        ...shown,
        {
          key: `${m.key}:more`,
          ids: rest,
          layer: STACK_CAP,
          label: `${plural(rest.length, "more marker")} near ${at}`,
          overflow: rest.length,
        },
      ];
}

export function stackHeightPx(placed: ReadonlyArray<PlacedMarker>): number {
  const layers = Math.max(
    0,
    ...placed.map((m) =>
      m.shape === "diamond" && !m.late ? Math.min(m.ids.length, STACK_CAP + 1) - 1 : 0,
    ),
  );
  return layers * STACK_STEP_PX;
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
  const [ownerId, setOwnerId] = useState<string | null>(null);
  const { activeId, onActiveChange, renderPopover } = markers;
  const labels = new Map(markers.items.map((item) => [item.id, item.label]));

  return (
    <div
      className="pointer-events-none absolute top-0"
      style={{ left: insetLeft, right: insetRight, height: bandHeight }}
    >
      {placed.flatMap((m) =>
        markerSlots(m, labels).map((slot) => {
          const isActive = activeId !== null && slot.ids.includes(activeId);
          const stacked = slot.layer > 0 || slot.key !== m.key;
          return (
            <HoverCard
              key={slot.key}
              open={isActive && ownerId === slot.key}
              onOpenChange={(open) => {
                if (open) {
                  setOwnerId(slot.key);
                  onActiveChange(slot.ids.at(-1) ?? null);
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
                aria-label={slot.label}
                data-active={isActive || undefined}
                className={cn(
                  "pointer-events-auto absolute -translate-x-1/2 translate-y-1/2 rounded-full p-0",
                  stacked &&
                    "size-2.5 min-w-0 rounded-xs before:absolute before:inset-y-0 before:-inset-x-1.5",
                )}
                style={{ left: `${m.leftPct}%`, bottom: slot.layer * STACK_STEP_PX }}
              >
                {slot.overflow > 0 ? (
                  <span className="text-3xs leading-none font-medium text-error-11">
                    +{slot.overflow}
                  </span>
                ) : (
                  <MarkerGlyph
                    shape={m.shape}
                    className={cn(
                      "transition-[scale,background-color,border-color] duration-150 ease-out group-hover:scale-125 group-data-active:scale-125 motion-reduce:transition-none",
                      m.late && "border-dashed",
                    )}
                  />
                )}
              </HoverCardTrigger>
              <HoverCardContent
                side="top"
                sideOffset={6}
                className="w-72 p-3 motion-reduce:transition-none"
              >
                <div className="flex max-h-80 flex-col divide-y divide-grayA-4 overflow-y-auto [&>*:not(:first-child)]:pt-3 [&>*:not(:last-child)]:pb-3">
                  {slot.ids.map((id) => (
                    <div key={id}>{renderPopover(id)}</div>
                  ))}
                </div>
              </HoverCardContent>
            </HoverCard>
          );
        }),
      )}
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
  const activeShape = markers.items.find((m) => m.id === markers.activeId)?.shape ?? "dot";
  return (
    <>
      {span && (
        <ReferenceArea
          x1={span[0]}
          x2={span[1]}
          fill={SHAPE_STYLE[activeShape].band}
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
            markers.activeId !== null && m.ids.includes(markers.activeId)
              ? SHAPE_STYLE[m.shape].activeLine
              : SHAPE_STYLE[m.shape].line
          }
          strokeDasharray="2 3"
          ifOverflow="hidden"
        />
      ))}
    </>
  );
}

function markerLabel(m: PlacedMarker): string {
  const at = new Date(m.atMs).toLocaleString();
  if (m.late) {
    return `${plural(m.ids.length, "marker")} after the last data point`;
  }
  if (m.ids.length > 1) {
    return `${plural(m.ids.length, "marker")} near ${at}`;
  }
  return `${m.label ?? "Marker"} at ${at}`;
}

export function MarkerGlyph({
  shape,
  className,
  style,
}: {
  shape: MarkerShape;
  className?: string;
  style?: CSSProperties;
}) {
  return (
    <span
      aria-hidden="true"
      className={cn("shrink-0 border", SHAPE_STYLE[shape].mark, className)}
      style={style}
    />
  );
}

"use client";

import { Button, HoverCard, HoverCardContent, HoverCardTrigger } from "@unkey/ui";
import type { ReactNode } from "react";
import { ReferenceArea } from "recharts";
import type { Domain } from "./chart-domain";
import { type ActiveChange, type ActiveMarker, useHoverOwner } from "./use-hover-owner";

export type ChartIncident = {
  id: string;
  fromMs: number;
  toMs: number;
  label: string;
  content: ReactNode;
};

export type ChartIncidents = {
  items: ChartIncident[];
  combine: (ids: string[]) => ChartIncident;
  active: ActiveMarker;
  onActiveChange: ActiveChange;
};

type PlacedIncident = {
  incident: ChartIncident;
  leftPct: number;
  widthPct: number;
  range: Domain;
  active: boolean;
};

const MIN_INCIDENT_PX = 4;

export const INCIDENT_FILL = "var(--color-errorA-4)";

function pctOf(ms: number, [start, end]: Domain): number {
  return Math.min(100, Math.max(0, ((ms - start) / (end - start)) * 100));
}

export function placeIncidents(
  { items, combine }: Pick<ChartIncidents, "items" | "combine">,
  activeId: string | null,
  domain: Domain,
  plotWidthPx: number,
): PlacedIncident[] {
  const [start, end] = domain;
  if (end <= start || plotWidthPx <= 0) {
    return [];
  }
  const minPct = (MIN_INCIDENT_PX / plotWidthPx) * 100;
  const msAt = (pct: number) => start + (pct / 100) * (end - start);
  const late = items.filter((incident) => incident.fromMs > end);
  const shown =
    late.length > 1
      ? [...items.filter((incident) => incident.fromMs <= end), combine(late.map((i) => i.id))]
      : items;
  return shown
    .filter((incident) => incident.toMs >= start)
    .map((incident) => {
      const from = pctOf(incident.fromMs, domain);
      const to = pctOf(incident.toMs, domain);
      const widthPct = Math.min(Math.max(to - from, minPct), 100);
      const leftPct = Math.min(Math.max(from - (widthPct - (to - from)) / 2, 0), 100 - widthPct);
      return {
        incident,
        leftPct,
        widthPct,
        range: [msAt(leftPct), msAt(leftPct + widthPct)],
        active: incident.id === activeId,
      };
    });
}

export function IncidentLayer({ placed }: { placed: PlacedIncident[] }) {
  return placed.map(({ incident, range, active }) => (
    <ReferenceArea
      key={incident.id}
      x1={range[0]}
      x2={range[1]}
      fill={active ? "var(--color-errorA-7)" : INCIDENT_FILL}
      fillOpacity={1}
      strokeOpacity={0}
    />
  ));
}

export function IncidentOverlay({
  incidents,
  placed,
  insetLeft,
  insetRight,
  insetTop,
  insetBottom,
}: {
  incidents: ChartIncidents;
  placed: PlacedIncident[];
  insetLeft: number;
  insetRight: number;
  insetTop: number;
  insetBottom: number;
}) {
  const hover = useHoverOwner(
    incidents.active,
    incidents.onActiveChange,
    placed.map((p) => p.incident.id),
  );
  return (
    <div
      className="pointer-events-none absolute"
      style={{ left: insetLeft, right: insetRight, top: insetTop, bottom: insetBottom }}
    >
      {placed.map(({ incident, active, leftPct, widthPct }) => (
        <HoverCard key={incident.id} {...hover(incident.id, incident.id)}>
          <HoverCardTrigger
            delay={0}
            closeDelay={100}
            render={<Button variant="ghost" />}
            aria-label={incident.label}
            data-active={active || undefined}
            className="pointer-events-auto absolute inset-y-0 h-auto min-w-0 rounded-none bg-transparent p-0 hover:bg-transparent focus:ring-0 focus-visible:ring-1 focus-visible:ring-error-9"
            style={{ left: `${leftPct}%`, width: `${widthPct}%` }}
          />
          <HoverCardContent
            side="bottom"
            sideOffset={6}
            collisionAvoidance={{ side: "none" }}
            className="w-72 p-3 motion-reduce:transition-none"
          >
            {incident.content}
          </HoverCardContent>
        </HoverCard>
      ))}
    </div>
  );
}

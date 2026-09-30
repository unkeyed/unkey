"use client";

import {
  Badge,
  FanOut,
  Node,
  NodeHeader,
  SpecGrid,
  SpecItem,
} from "@/app/(app)/[workspaceSlug]/projects/_components/canvas/nodes";
import { RectFlag } from "@/app/(app)/[workspaceSlug]/projects/_components/region/rect-flag";
import { regionInfo } from "@/app/(app)/[workspaceSlug]/projects/_components/region/region-info";
import { Logomark } from "@/components/logomark";
import {
  IconEarthOutline18,
  IconHardDriveOutline18,
  IconLayers3Outline18,
  IconMicrochipOutline18,
  IconRamOutline18,
  IconSitemapOutline18,
} from "@unkey/icons";
import { cn } from "cn";
import { useState } from "react";
import {
  type ComputeDraft,
  type RegionDraft,
  formatCpu,
  formatMemory,
  formatReplicas,
  formatStorage,
  sizeLabel,
} from "./model";

function rangeOf(region: RegionDraft): string {
  return formatReplicas({ kind: "uniform", min: region.replicasMin, max: region.replicasMax });
}

type View = "map" | "nodes";

type VizProps = {
  draft: ComputeDraft;
  unavailable: ReadonlySet<string>;
  hovered: string | null;
  onHover: (name: string | null) => void;
};

const LON_MIN = -170;
const LAT_MAX = 78;
const MAP_W = 350;
const MAP_H = 128;
const STEP = 3.5;

type Ring = ReadonlyArray<readonly [number, number]>;

const LAND: readonly Ring[] = [
  [
    [-165, 68],
    [-140, 70],
    [-95, 72],
    [-80, 68],
    [-62, 60],
    [-55, 50],
    [-66, 44],
    [-76, 35],
    [-81, 25],
    [-83, 29],
    [-97, 26],
    [-97, 20],
    [-87, 21],
    [-83, 10],
    [-78, 8],
    [-92, 15],
    [-105, 20],
    [-112, 30],
    [-118, 34],
    [-124, 40],
    [-124, 48],
    [-135, 58],
    [-150, 60],
    [-165, 60],
  ],
  [
    [-55, 80],
    [-20, 80],
    [-20, 70],
    [-45, 60],
    [-55, 70],
  ],
  [
    [-78, 8],
    [-60, 10],
    [-50, 0],
    [-35, -7],
    [-40, -22],
    [-48, -28],
    [-58, -38],
    [-65, -42],
    [-68, -55],
    [-75, -50],
    [-72, -30],
    [-70, -18],
    [-80, -5],
  ],
  [
    [-10, 36],
    [-9, 44],
    [-2, 44],
    [-5, 48],
    [2, 51],
    [8, 54],
    [10, 58],
    [5, 62],
    [15, 70],
    [30, 71],
    [40, 68],
    [40, 45],
    [28, 41],
    [20, 40],
    [12, 38],
    [3, 43],
  ],
  [
    [-6, 50],
    [2, 51],
    [0, 54],
    [-3, 58],
    [-6, 56],
  ],
  [
    [-17, 21],
    [-10, 30],
    [-5, 36],
    [10, 37],
    [20, 32],
    [32, 31],
    [35, 28],
    [43, 12],
    [51, 12],
    [42, -2],
    [40, -15],
    [35, -25],
    [20, -35],
    [15, -28],
    [12, -15],
    [9, -2],
    [5, 5],
    [-8, 5],
    [-17, 14],
  ],
  [
    [40, 45],
    [40, 68],
    [60, 70],
    [80, 73],
    [110, 76],
    [140, 72],
    [180, 68],
    [178, 62],
    [160, 58],
    [142, 52],
    [135, 42],
    [128, 35],
    [122, 30],
    [120, 22],
    [108, 20],
    [105, 10],
    [100, 2],
    [98, 12],
    [92, 22],
    [88, 22],
    [80, 10],
    [77, 8],
    [72, 20],
    [66, 25],
    [57, 25],
    [55, 17],
    [45, 13],
    [35, 30],
    [35, 37],
    [27, 38],
    [27, 42],
  ],
  [
    [130, 31],
    [141, 35],
    [142, 45],
    [139, 40],
    [132, 34],
  ],
  [
    [95, 5],
    [105, -6],
    [115, -8],
    [125, -9],
    [140, -8],
    [140, -3],
    [120, 1],
    [108, 2],
  ],
  [
    [113, -22],
    [122, -18],
    [130, -12],
    [137, -12],
    [142, -11],
    [146, -19],
    [153, -26],
    [150, -37],
    [140, -38],
    [132, -32],
    [115, -34],
  ],
  [
    [172, -34],
    [178, -38],
    [174, -42],
    [167, -46],
    [170, -44],
  ],
];

function inside(lon: number, lat: number, ring: Ring): boolean {
  let hit = false;
  for (let i = 0, j = ring.length - 1; i < ring.length; j = i++) {
    const [xi, yi] = ring[i] ?? [0, 0];
    const [xj, yj] = ring[j] ?? [0, 0];
    if (yi > lat !== yj > lat && lon < ((xj - xi) * (lat - yi)) / (yj - yi) + xi) {
      hit = !hit;
    }
  }
  return hit;
}

const DOTS: ReadonlyArray<{ x: number; y: number }> = (() => {
  const dots: { x: number; y: number }[] = [];
  for (let y = STEP / 2; y < MAP_H; y += STEP) {
    for (let x = STEP / 2; x < MAP_W; x += STEP) {
      if (LAND.some((ring) => inside(x + LON_MIN, LAT_MAX - y, ring))) {
        dots.push({ x, y });
      }
    }
  }
  return dots;
})();

const VIEWS = [
  { value: "map", label: "Map", Icon: IconEarthOutline18 },
  { value: "nodes", label: "Nodes", Icon: IconSitemapOutline18 },
] as const;

export function RegionViz(props: VizProps) {
  const [view, setView] = useState<View>("nodes");
  return (
    <div className="relative aspect-[350/128] w-full overflow-hidden bg-grayA-2">
      <div className="absolute top-2 right-2 z-10 flex rounded-md border border-grayA-4 bg-raised p-0.5">
        {VIEWS.map(({ value, label, Icon }) => (
          <button
            key={value}
            type="button"
            aria-pressed={view === value}
            aria-label={label}
            title={label}
            onClick={() => setView(value)}
            className={cn(
              "flex size-6 items-center justify-center rounded text-gray-11 transition-colors hover:text-gray-12",
              view === value && "bg-grayA-3 text-gray-12",
            )}
          >
            <Icon className="size-3.5" />
          </button>
        ))}
      </div>
      {view === "map" ? <MapView {...props} /> : <NodesView {...props} />}
    </div>
  );
}

const UNAVAILABLE = "Unavailable for scheduling";

function MapView({ draft, unavailable, hovered, onHover }: VizProps) {
  const regions = draft.regions.map((r) => ({ ...regionInfo(r.name), count: rangeOf(r) }));
  const offMap = regions.filter((r) => !r.pin);
  return (
    <>
      <div className="absolute top-1/2 right-0 left-0 aspect-[350/128] -translate-y-1/2">
        <svg
          viewBox={`0 0 ${MAP_W} ${MAP_H}`}
          className="absolute inset-0 size-full"
          aria-hidden="true"
        >
          {DOTS.map((d) => (
            <circle key={`${d.x}-${d.y}`} cx={d.x} cy={d.y} r={0.85} className="fill-gray-6" />
          ))}
        </svg>
        {regions.map((region) => {
          if (!region.pin) {
            return null;
          }
          const left = ((region.pin.lon - LON_MIN) / MAP_W) * 100;
          const top = ((LAT_MAX - region.pin.lat) / MAP_H) * 100;
          const active = hovered === region.name;
          const blocked = unavailable.has(region.name);
          return (
            <div
              key={region.name}
              className="absolute"
              style={{ left: `${left}%`, top: `${top}%` }}
              onMouseEnter={() => onHover(region.name)}
              onMouseLeave={() => onHover(null)}
            >
              <span
                className={cn(
                  "absolute -translate-x-1/2 -translate-y-1/2 animate-pop rounded-full ring-4 transition-[width,height] duration-150 motion-reduce:animate-none",
                  blocked ? "bg-warning-9 ring-warning-4" : "bg-gray-12 ring-grayA-4",
                  active ? "size-3" : "size-2",
                )}
              />
              <span className="absolute size-6 -translate-x-1/2 -translate-y-1/2" />
              {active ? (
                <span
                  className={cn(
                    "pointer-events-none absolute top-0 flex -translate-y-1/2 animate-pop items-center gap-1.5 whitespace-nowrap rounded-md border bg-raised px-1.5 py-0.5 font-mono text-[11px] text-gray-12 shadow-sm motion-reduce:animate-none",
                    region.pin.label === "left" ? "right-3" : "left-3",
                  )}
                >
                  <RectFlag flag={region.flag} size="sm" />
                  {region.name}
                  <span className="text-gray-10">×{region.count}</span>
                  {blocked ? <span className="text-warning-11">{UNAVAILABLE}</span> : null}
                </span>
              ) : null}
            </div>
          );
        })}
      </div>
      {offMap.length > 0 ? (
        <div className="absolute bottom-2 left-2 flex gap-1.5">
          {offMap.map((region) => (
            <span
              key={region.name}
              onMouseEnter={() => onHover(region.name)}
              onMouseLeave={() => onHover(null)}
              className={cn(
                "flex items-center gap-1.5 rounded-md border bg-raised px-1.5 py-0.5 font-mono text-[11px] text-gray-11",
                hovered === region.name ? "border-grayA-8" : "border-grayA-4",
              )}
            >
              <RectFlag flag={region.flag} size="sm" />
              {region.name} · not on map
            </span>
          ))}
        </div>
      ) : null}
    </>
  );
}

const NODE_W = 208;
const NODE_GAP = 24;

function NodesView({ draft, unavailable, hovered, onHover }: VizProps) {
  const regions = draft.regions.map((r) => ({ ...regionInfo(r.name), count: rangeOf(r) }));
  const n = Math.max(regions.length, 1);
  const specs = [
    ["CPU", IconMicrochipOutline18, formatCpu(draft.cpuMillicores)],
    ["RAM", IconRamOutline18, formatMemory(draft.memoryMib)],
    ["Disk", IconHardDriveOutline18, formatStorage(draft.storageMib)],
  ] as const;
  return (
    <div className="flex h-full flex-col justify-center px-6 py-5">
      <div
        className="mx-auto flex w-full flex-col"
        style={{ maxWidth: n * NODE_W + (n - 1) * NODE_GAP }}
      >
        {regions.length > 1 ? (
          <>
            <div className="mx-auto flex items-center gap-2 rounded-lg bg-raised px-3 py-2 shadow-sm ring-1 ring-grayA-5">
              <Logomark className="size-6 rounded-md bg-grayA-3" />
              <span className="flex flex-col">
                <span className="text-xs font-medium text-gray-12">Unkey edge</span>
                <span className="text-[11px] text-gray-10">Routes to the nearest region</span>
              </span>
            </div>
            <FanOut targets={regions.length} />
          </>
        ) : null}
        <div
          className="grid"
          style={{ gridTemplateColumns: `repeat(${n}, minmax(0, 1fr))`, columnGap: NODE_GAP }}
        >
          {regions.map((region) => {
            const blocked = unavailable.has(region.name);
            return (
              <Node
                key={region.name}
                edge="ring"
                tone={blocked ? "warning" : "default"}
                active={hovered === region.name}
                onMouseEnter={() => onHover(region.name)}
                onMouseLeave={() => onHover(null)}
                title={blocked ? UNAVAILABLE : undefined}
              >
                <NodeHeader
                  leading={<RectFlag flag={region.flag} size="sm" />}
                  title={region.city}
                  meta={region.name}
                  right={
                    <>
                      {blocked ? <Badge tone="warning">Unavailable</Badge> : null}
                      <Badge>{sizeLabel(draft)}</Badge>
                    </>
                  }
                />
                <SpecGrid>
                  <SpecItem
                    icon={<IconLayers3Outline18 />}
                    label="Instances"
                    value={region.count}
                  />
                  {specs.map(([label, Icon, value]) => (
                    <SpecItem key={label} icon={<Icon />} label={label} value={value} />
                  ))}
                </SpecGrid>
              </Node>
            );
          })}
        </div>
      </div>
    </div>
  );
}

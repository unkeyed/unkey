"use client";

import { EdgeNode, Spec, sizeSpecs } from "@/components/compute-diagram";
import { FanIn } from "@/components/fan-in";
import { FanOut } from "@/components/fan-out";
import { RegionFlag } from "@/components/region-flag";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import {
  type InstanceState,
  instanceLabel,
  runningCountText,
} from "@/lib/collections/deploy/instance-status";
import { sizeLabel } from "@/lib/compute/sizing";
import { routes } from "@/lib/navigation/routes";
import { regionInfo } from "@/lib/regions";
import { IconLayers3Outline18, IconServerOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import { Badge, Popover, PopoverContent, PopoverTrigger } from "@unkey/ui";
import { cn } from "cn";
import Link from "next/link";
import { useId, useMemo, useState } from "react";
import { useDeployment } from "../layout-provider";
import {
  type DiagramInstance,
  type EnteredTrack,
  type RegionGroup,
  resolveInstancesView,
  trackEntered,
} from "./instances-model";
import type { StepResolution } from "./stages-model";

type Density = "comfortable" | "compact";

const densityOf = (regions: number): Density => (regions > 3 ? "compact" : "comfortable");

// The row never wraps: it keeps at least minW per region and the panel scrolls
// sideways below that, so the edge and fan-out always line up with the cards.
const densityLayout: Record<
  Density,
  { minW: number; maxW: number; gap: number; gapClass: string; fanOut: string; regionCode: string }
> = {
  comfortable: {
    minW: 260,
    maxW: 320,
    gap: 32,
    gapClass: "gap-x-8",
    fanOut: "-mx-4 w-[calc(100%+2rem)]",
    regionCode: "truncate",
  },
  compact: {
    minW: 132,
    maxW: 220,
    gap: 12,
    gapClass: "gap-x-3",
    fanOut: "-mx-1.5 w-[calc(100%+0.75rem)]",
    regionCode: "sr-only",
  },
};

const instanceDensity: Record<Density, { row: string; tile: TileVariant }> = {
  comfortable: { row: "gap-2.5 py-2 pr-2.5 pl-2", tile: "instance" },
  compact: { row: "gap-2 py-1.5 pr-2 pl-1.5", tile: "instanceCompact" },
};

type TileVariant = "region" | "instance" | "instanceCompact";

const tileTone: Record<InstanceState, Record<Exclude<TileVariant, "instanceCompact">, string>> = {
  running: {
    region: "bg-successA-4 text-success-11 ring-1 ring-inset ring-successA-5",
    instance: "bg-successA-3 text-success-11",
  },
  starting: {
    region: "bg-grayA-3 text-gray-11",
    instance: "bg-grayA-3 text-gray-11",
  },
  unhealthy: {
    region: "bg-errorA-4 text-error-11 ring-1 ring-inset ring-errorA-5",
    instance:
      "bg-errorA-4 text-error-11 after:absolute after:-inset-[3px] after:rounded-lg after:border-[1.5px] after:border-errorA-7 motion-safe:after:animate-ring-pulse",
  },
  inactive: {
    region: "bg-grayA-3 text-gray-9 opacity-40",
    instance: "bg-grayA-3 text-gray-9 opacity-40",
  },
};

const tileSize: Record<TileVariant, { tile: string; icon: string }> = {
  region: { tile: "size-6 rounded-[5px]", icon: "size-3.5" },
  instance: {
    tile: "size-7 rounded-md motion-safe:group-data-[entered=running]/card:animate-tile-settle",
    icon: "size-4",
  },
  instanceCompact: {
    tile: "size-6 rounded-[5px] motion-safe:group-data-[entered=running]/card:animate-tile-settle",
    icon: "size-3.5",
  },
};

const neutralCard = { card: "bg-gray-2 ring-grayA-3", meta: "text-gray-10" };

const cardTone: Record<InstanceState, { card: string; meta: string }> = {
  running: neutralCard,
  starting: neutralCard,
  unhealthy: { card: "bg-error-2 ring-error-7", meta: "text-error-11" },
  inactive: neutralCard,
};

export function InstancesDiagram({ stage }: { stage: StepResolution["status"] }) {
  const { deployment } = useDeployment();
  const { instances, desiredRegions, targetRegions, desiredState, desiredInstanceCount } =
    deployment;
  const view = useMemo(
    () =>
      resolveInstancesView({
        instances,
        desiredRegions,
        targetRegions,
        desiredState,
        desiredInstanceCount,
        stage,
      }),
    [instances, desiredRegions, targetRegions, desiredState, desiredInstanceCount, stage],
  );

  if (view.kind === "empty") {
    return (
      <div className="px-8 py-8">
        <p className="mx-auto max-w-80 rounded-lg bg-raised px-3 py-6 text-center text-xs text-gray-10 shadow-sm ring-1 ring-grayA-5">
          {view.message}
        </p>
      </div>
    );
  }

  const n = view.regions.length;
  const density = densityOf(n);
  const layout = densityLayout[density];
  const rowWidth = (w: number) => n * w + (n - 1) * layout.gap;
  return (
    <div className="overflow-x-auto px-8 py-8">
      <FanIn className="-mt-8 mx-auto h-33 w-full max-w-120" />
      <div
        className="mx-auto flex flex-col motion-safe:animate-upgrade-rise"
        style={{ minWidth: rowWidth(layout.minW), maxWidth: rowWidth(layout.maxW) }}
      >
        <EdgeNode />
        <FanOut targets={n} className={cn("h-13", layout.fanOut)} />
        <div
          className={cn("grid items-start", layout.gapClass)}
          style={{ gridTemplateColumns: `repeat(${n}, minmax(0, 1fr))` }}
        >
          {view.regions.map((region) => (
            <RegionCard key={region.key} region={region} density={density} />
          ))}
        </div>
      </div>
    </div>
  );
}

function slots(count: number, prefix: string): string[] {
  return Array.from({ length: count }, (_, i) => `${prefix}-${i}`);
}

function busyLabel(content: RegionGroup["content"]): string | null {
  return match(content)
    .with({ kind: "instances" }, ({ waiting }) => (waiting > 0 ? "Waiting for instances" : null))
    .with({ kind: "waiting" }, () => "Waiting for instances")
    .with({ kind: "skeleton" }, () => "Loading instances")
    .exhaustive();
}

function RegionCard({ region, density }: { region: RegionGroup; density: Density }) {
  const nameId = useId();
  const busy = busyLabel(region.content);
  return (
    <div
      role="group"
      aria-labelledby={region.name ? `${nameId}-city ${nameId}-code` : nameId}
      data-state={region.state}
      className="flex min-w-0 flex-col rounded-lg bg-raised shadow-sm ring-1 ring-grayA-5 data-[state=unhealthy]:ring-error-7"
    >
      <div className="flex min-w-0 items-center gap-1.5 px-3 py-2.5">
        {region.name ? (
          <RegionName
            id={nameId}
            name={region.name}
            codeClassName={densityLayout[density].regionCode}
          />
        ) : (
          <RegionNamePlaceholder id={nameId} />
        )}
        <span className="ml-auto flex shrink-0 items-center gap-2">
          <SizeBadge region={region} />
          <HealthTile state={region.state} variant="region" />
          <span className="sr-only">{instanceLabel[region.state]}</span>
        </span>
      </div>
      <ul aria-busy={busy ? true : undefined} className="flex flex-col gap-1.5 px-2.5 pb-2.5">
        {busy ? <li className="sr-only">{busy}</li> : null}
        {match(region.content)
          .with({ kind: "instances" }, ({ instances, waiting }) => (
            <>
              {instances.map((instance) => (
                <InstanceCard key={instance.id} instance={instance} density={density} />
              ))}
              {slots(waiting, "waiting").map((slot) => (
                <PlaceholderCard key={slot} density={density} label="Waiting for instance" />
              ))}
            </>
          ))
          .with({ kind: "waiting" }, () => (
            <PlaceholderCard density={density} label="Waiting for instances" />
          ))
          .with({ kind: "skeleton" }, ({ count }) =>
            slots(count, "slot").map((slot) => <PlaceholderCard key={slot} density={density} />),
          )
          .exhaustive()}
      </ul>
    </div>
  );
}

function RegionName({
  id,
  name,
  codeClassName,
}: { id: string; name: string; codeClassName: string }) {
  return (
    <>
      <RegionFlag region={name} shape="rect" size="sm" />
      <span id={`${id}-city`} className="truncate text-xs font-medium text-gray-12">
        {regionInfo(name).city}
      </span>
      <span id={`${id}-code`} className={cn("font-mono text-2xs text-gray-10", codeClassName)}>
        ({name})
      </span>
    </>
  );
}

function RegionNamePlaceholder({ id }: { id: string }) {
  return (
    <>
      <span className="h-3.5 w-5 shrink-0 rounded-xs bg-grayA-3" />
      <span className="text-xs">
        <span className="inline-block h-2.5 w-28 rounded-sm bg-grayA-3 align-middle" />
      </span>
      <span id={id} className="sr-only">
        Region not assigned yet
      </span>
    </>
  );
}

function SizeBadge({ region }: { region: RegionGroup }) {
  const { deployment } = useDeployment();
  const label = sizeLabel(deployment);
  return (
    <Popover>
      <PopoverTrigger
        openOnHover
        delay={0}
        closeDelay={100}
        aria-label={`${label} instance size, details`}
        className="rounded-md focus-visible:outline-hidden focus-visible:ring-1 focus-visible:ring-grayA-12"
      >
        <Badge
          variant="code"
          className="inline-grid h-6 min-w-6 cursor-help place-items-center px-1.5"
        >
          {label}
        </Badge>
      </PopoverTrigger>
      <PopoverContent
        side="bottom"
        align="end"
        className="flex w-50 flex-col gap-1.5 px-2.5 py-2 font-mono text-3xs"
      >
        <span className="font-sans text-2xs font-medium">{label}</span>
        <Spec icon={<IconLayers3Outline18 />} label="Instances" value={runningCountText(region)} />
        <span className="font-sans text-gray-10">Per instance</span>
        <span className="grid grid-cols-2 gap-x-3 gap-y-1 whitespace-nowrap">
          {sizeSpecs(deployment).map(({ label, Icon, value }) => (
            <Spec key={label} icon={<Icon />} label={label} value={value} />
          ))}
        </span>
      </PopoverContent>
    </Popover>
  );
}

function HealthTile({ state, variant }: { state: InstanceState; variant: TileVariant }) {
  const size = tileSize[variant];
  return (
    <span
      aria-hidden="true"
      className={cn(
        "relative grid shrink-0 place-items-center transition-[background-color,color,opacity] duration-200 ease-in-out motion-reduce:transition-none",
        size.tile,
        tileTone[state][variant === "region" ? "region" : "instance"],
      )}
    >
      {state === "starting" ? (
        <>
          <IconServerOutline18 className={cn("text-gray-9 opacity-55", size.icon)} />
          <IconServerOutline18
            className={cn(
              "absolute text-gray-12 [mask-image:linear-gradient(100deg,transparent_30%,#000_50%,transparent_70%)] [mask-size:250%_100%] motion-safe:animate-icon-shimmer motion-reduce:hidden",
              size.icon,
            )}
          />
        </>
      ) : (
        <IconServerOutline18 className={size.icon} />
      )}
    </span>
  );
}

function InstanceCard({ instance, density }: { instance: DiagramInstance; density: Density }) {
  const entered = useEnteredState(instance);
  const tone = cardTone[instance.state];
  return (
    <li
      data-entered={entered ?? undefined}
      className={cn(
        "group/card transition-[background-color,box-shadow] duration-200 ease-in-out motion-safe:data-[entered=running]:animate-wash-success motion-safe:data-[entered=unhealthy]:animate-wash-error motion-reduce:transition-none",
        "flex min-w-0 items-center rounded-md ring-1 ring-inset",
        instanceDensity[density].row,
        tone.card,
      )}
    >
      <HealthTile state={instance.state} variant={instanceDensity[density].tile} />
      <span className="flex min-w-0 flex-col gap-px">
        <span className="truncate text-2xs font-medium text-gray-12">{instance.id}</span>
        <span className={cn("truncate text-3xs tabular-nums", tone.meta)}>
          {instance.detail ? (
            <>
              <span className="sr-only">{instanceLabel[instance.state]}: </span>
              {instance.detail}
            </>
          ) : (
            instanceLabel[instance.state]
          )}
        </span>
      </span>
      {instance.state === "unhealthy" ? <InstanceLogsLink instanceId={instance.id} /> : null}
    </li>
  );
}

function InstanceLogsLink({ instanceId }: { instanceId: string }) {
  const workspace = useWorkspaceNavigation();
  const { deployment } = useDeployment();
  return (
    <Link
      href={routes.projects.logs({
        workspaceSlug: workspace.slug,
        projectId: deployment.projectId,
        deploymentId: deployment.id,
        instanceId,
      })}
      aria-label={`Logs for ${instanceId}`}
      className="-m-1 ml-auto shrink-0 rounded-sm p-1 text-3xs font-medium text-gray-11 transition-colors hover:text-gray-12 focus-visible:outline-hidden focus-visible:ring-1 focus-visible:ring-grayA-12"
    >
      Logs
    </Link>
  );
}

function PlaceholderCard({ label, density }: { label?: string; density: Density }) {
  return (
    <li
      aria-hidden="true"
      className={cn(
        "flex min-w-0 items-center rounded-md ring-1 ring-inset",
        instanceDensity[density].row,
        cardTone.starting.card,
      )}
    >
      <HealthTile state="starting" variant={instanceDensity[density].tile} />
      <span className="flex min-w-0 flex-col gap-px">
        <span className="truncate text-2xs font-medium text-gray-11">
          {label ?? <span className="inline-block h-2 w-24 rounded-sm bg-grayA-3 align-middle" />}
        </span>
        <span className="text-3xs">
          <span className="inline-block h-1.5 w-12 rounded-sm bg-grayA-3 align-middle" />
        </span>
      </span>
    </li>
  );
}

function useEnteredState(instance: DiagramInstance): InstanceState | null {
  const [tracked, setTracked] = useState<EnteredTrack>({
    state: instance.state,
    restarts: instance.restarts,
    entered: null,
  });
  const next = trackEntered(tracked, instance);
  if (next !== tracked) {
    setTracked(next);
  }
  return next.entered;
}

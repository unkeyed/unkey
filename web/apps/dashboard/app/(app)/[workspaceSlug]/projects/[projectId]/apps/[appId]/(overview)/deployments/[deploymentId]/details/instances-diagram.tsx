"use client";

import { EdgeNode, Spec, sizeSpecs } from "@/components/compute-diagram";
import { FanIn } from "@/components/fan-in";
import { FanOut } from "@/components/fan-out";
import { RegionFlag } from "@/components/region-flag";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { type InstanceState, instanceLabel } from "@/lib/collections/deploy/instance-status";
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
import { runningCountText } from "./deployment-health";
import {
  type DiagramInstance,
  type EdgeRouting,
  type EnteredTrack,
  type RegionGroup,
  resolveInstancesView,
  trackEntered,
} from "./instances-model";
import type { StepResolution } from "./stages-model";

const REGION_W = 320;
const REGION_GAP = 32;

const sideBySideAt: Partial<Record<number, { edge: string; grid: string }>> = {
  2: {
    edge: "@min-[552px]/diagram:flex",
    grid: "@min-[552px]/diagram:grid-cols-2",
  },
  3: {
    edge: "@min-[844px]/diagram:flex",
    grid: "@min-[844px]/diagram:grid-cols-3",
  },
  4: {
    edge: "@min-[1136px]/diagram:flex",
    grid: "@min-[1136px]/diagram:grid-cols-4",
  },
  5: {
    edge: "@min-[1428px]/diagram:flex",
    grid: "@min-[1428px]/diagram:grid-cols-5",
  },
};

type TileVariant = "region" | "instance";

const tileTone: Record<InstanceState, Record<TileVariant, string>> = {
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
};

const neutralCard = { card: "bg-gray-2 ring-grayA-3", meta: "text-gray-10" };

const cardTone: Record<InstanceState, { card: string; meta: string }> = {
  running: neutralCard,
  starting: neutralCard,
  unhealthy: { card: "bg-error-2 ring-error-7", meta: "text-error-11" },
  inactive: neutralCard,
};

function edgeSubtitle(routing: EdgeRouting): string {
  return match(routing)
    .with({ kind: "build" }, () => "Waiting for build")
    .with({ kind: "waiting" }, () => "Waiting for a healthy instance")
    .with({ kind: "stopped" }, () => "Not serving traffic")
    .with({ kind: "partial" }, ({ live, total }) => `Routing to ${live} of ${total} regions`)
    .with({ kind: "all" }, () => "Routes to the nearest region")
    .exhaustive();
}

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
  const layout = sideBySideAt[n];
  return (
    <div className="@container/diagram px-8 py-8">
      {layout ? (
        <FanIn className={cn("-mt-8 mx-auto hidden h-33 w-full max-w-120", layout.edge)} />
      ) : null}
      <div
        className="mx-auto flex w-full flex-col motion-safe:animate-diagram-rise"
        style={{ maxWidth: n * REGION_W + (n - 1) * REGION_GAP }}
      >
        {layout ? (
          <div className={cn("hidden flex-col", layout.edge)}>
            <EdgeNode subtitle={edgeSubtitle(view.routing)} />
            <FanOut targets={n} className="-mx-4 h-13 w-[calc(100%+2rem)]" />
          </div>
        ) : null}
        <div className={cn("grid grid-cols-1 items-start gap-x-8 gap-y-4", layout?.grid)}>
          {view.regions.map((region) => (
            <RegionCard key={region.key} region={region} />
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

function RegionCard({ region }: { region: RegionGroup }) {
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
          <RegionName id={nameId} name={region.name} />
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
                <InstanceCard key={instance.id} instance={instance} />
              ))}
              {slots(waiting, "waiting").map((slot) => (
                <PlaceholderCard key={slot} label="Waiting for instance" />
              ))}
            </>
          ))
          .with({ kind: "waiting" }, () => <PlaceholderCard label="Waiting for instances" />)
          .with({ kind: "skeleton" }, ({ count }) =>
            slots(count, "slot").map((slot) => <PlaceholderCard key={slot} />),
          )
          .exhaustive()}
      </ul>
    </div>
  );
}

function RegionName({ id, name }: { id: string; name: string }) {
  return (
    <>
      <RegionFlag region={name} shape="rect" size="sm" />
      <span id={`${id}-city`} className="shrink-0 text-xs font-medium text-gray-12">
        {regionInfo(name).city}
      </span>
      <span id={`${id}-code`} className="truncate font-mono text-2xs text-gray-10">
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
          {sizeSpecs(deployment).map(([name, Icon, value]) => (
            <Spec key={name} icon={<Icon />} label={name} value={value} />
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
        tileTone[state][variant],
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

function InstanceCard({ instance }: { instance: DiagramInstance }) {
  const entered = useEnteredState(instance);
  const tone = cardTone[instance.state];
  return (
    <li
      data-entered={entered ?? undefined}
      className={cn(
        "group/card transition-[background-color,box-shadow] duration-200 ease-in-out motion-safe:data-[entered=running]:animate-wash-success motion-safe:data-[entered=unhealthy]:animate-wash-error motion-reduce:transition-none",
        "flex min-w-0 items-center gap-2.5 rounded-md py-2 pr-2.5 pl-2 ring-1 ring-inset",
        tone.card,
      )}
    >
      <HealthTile state={instance.state} variant="instance" />
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

function PlaceholderCard({ label }: { label?: string }) {
  return (
    <li
      aria-hidden="true"
      className={cn(
        "flex min-w-0 items-center gap-2.5 rounded-md py-2 pr-2.5 pl-2 ring-1 ring-inset",
        cardTone.starting.card,
      )}
    >
      <HealthTile state="starting" variant="instance" />
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

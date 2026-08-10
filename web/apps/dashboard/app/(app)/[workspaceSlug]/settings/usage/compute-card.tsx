"use client";

// Verified keys carry no per-project or per-app figure: ClickHouse counts distinct
// keys per workspace only, so the footer states [no data] for the project grain and
// shows the workspace count instead.

import { formatCompactQuantity, formatDollars, formatNumber } from "@/lib/fmt";
import { ChevronRight, Cube } from "@unkey/icons";
import {
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemFooter,
  ItemGroup,
  ItemHeader,
  ItemMedia,
  ItemSeparator,
  ItemTitle,
} from "@unkey/ui";
import { Fragment, useState } from "react";
import {
  type ComputeTree,
  type UsageApp,
  type UsageProject,
  type UsageQuantities,
  microCentsToDisplayCents,
} from "./compute-tree";

const COLUMNS =
  "grid min-w-[42rem] grid-cols-[minmax(8rem,1fr)_5rem_7rem_6rem_7rem_5rem] items-center gap-3 px-4";

const BAND =
  "border-grayA-4 border-y bg-grayA-2 py-2 font-semibold text-[10px] text-gray-9 uppercase tracking-wider";

const QUANTITY_COLUMNS: ReadonlyArray<{ key: keyof UsageQuantities; label: string }> = [
  { key: "cpuHours", label: "CPU hrs" },
  { key: "memoryGiBHours", label: "Memory GiB-hrs" },
  { key: "egressGiB", label: "Egress GiB" },
  { key: "diskGiBHours", label: "Disk GiB-hrs" },
];

type ComputeCardProps = {
  tree: ComputeTree;
  /**
   * Workspace gross for the period. It covers the verified keys the tree has no
   * grain for, so it is read from the workspace endpoint rather than summed here.
   * Null when that endpoint failed.
   */
  workspaceCents: number | null;
  activeKeys: number | null;
};

export function ComputeCard({ tree, workspaceCents, activeKeys }: ComputeCardProps) {
  const [open, setOpen] = useState<ReadonlySet<string>>(new Set());

  const toggle = (projectId: string) =>
    setOpen((current) => {
      const next = new Set(current);
      if (!next.delete(projectId)) {
        next.add(projectId);
      }
      return next;
    });

  return (
    <ItemGroup variant="outline">
      <ItemHeader>
        <ItemMedia className="bg-orangeA-3 text-orange-11">
          <Cube />
        </ItemMedia>
        <ItemContent>
          <ItemTitle>Compute</ItemTitle>
          <ItemDescription>Usage per project this period</ItemDescription>
        </ItemContent>
        <ItemActions className="font-medium text-sm tabular-nums">
          {workspaceCents === null ? (
            <span className="font-normal text-gray-9">Unavailable</span>
          ) : (
            formatDollars(workspaceCents)
          )}
        </ItemActions>
      </ItemHeader>
      {tree.projects.length === 0 ? (
        <>
          <ItemSeparator />
          <Item>
            <ItemContent>
              <ItemDescription>No compute usage recorded this period.</ItemDescription>
            </ItemContent>
          </Item>
        </>
      ) : (
        tree.projects.map((project) => (
          <Fragment key={project.projectId}>
            <ItemSeparator />
            <ProjectRow
              project={project}
              open={open.has(project.projectId)}
              onToggle={() => toggle(project.projectId)}
            />
          </Fragment>
        ))
      )}
      <ItemSeparator />
      <ItemFooter>
        Verified keys are counted per workspace, so a per-project figure is [no data].
        {activeKeys === null ? "" : ` This period: ${formatNumber(activeKeys)} keys.`}
      </ItemFooter>
    </ItemGroup>
  );
}

function ProjectRow({
  project,
  open,
  onToggle,
}: {
  project: UsageProject;
  open: boolean;
  onToggle: () => void;
}) {
  return (
    <div>
      <Item render={<button type="button" aria-expanded={open} onClick={onToggle} />}>
        <ChevronRight
          iconSize="sm-regular"
          className={`shrink-0 text-gray-9 transition-transform duration-150 motion-reduce:transition-none ${open ? "rotate-90" : ""}`}
        />
        <ItemContent>
          <ItemTitle>{project.name}</ItemTitle>
          <ItemDescription>
            {project.apps.length} {project.apps.length === 1 ? "app" : "apps"}
          </ItemDescription>
        </ItemContent>
        <ItemActions className="font-medium tabular-nums">
          {formatDollars(microCentsToDisplayCents(project.microCents))}
        </ItemActions>
      </Item>
      {open ? (
        <div className="overflow-x-auto pb-2">
          <div className={`${COLUMNS} ${BAND}`}>
            <div>App</div>
            {QUANTITY_COLUMNS.map((column) => (
              <div key={column.key} className="text-right">
                {column.label}
              </div>
            ))}
            <div className="text-right">Cost</div>
          </div>
          {project.apps.map((app) => (
            <AppRows key={app.appId} app={app} />
          ))}
        </div>
      ) : null}
    </div>
  );
}

function AppRows({ app }: { app: UsageApp }) {
  return (
    <div>
      <div className={`${COLUMNS} pt-2.5 pb-1`}>
        <span className="truncate font-medium text-[13px] text-gray-12">{app.name}</span>
        <Quantities quantities={app} className="text-[13px] text-gray-11" />
        <span className="text-right font-medium text-[13px] text-gray-12 tabular-nums">
          {formatDollars(microCentsToDisplayCents(app.microCents))}
        </span>
      </div>
      {app.environments.map((environment) => (
        <div key={environment.environmentId} className={`${COLUMNS} py-1 last:pb-2.5`}>
          <span className="truncate pl-3 text-gray-10 text-xs">{environment.name}</span>
          <Quantities quantities={environment} className="text-gray-10 text-xs" />
          <span className="text-right text-gray-11 text-xs tabular-nums">
            {formatDollars(microCentsToDisplayCents(environment.microCents))}
          </span>
        </div>
      ))}
    </div>
  );
}

function Quantities({ quantities, className }: { quantities: UsageQuantities; className: string }) {
  return (
    <>
      {QUANTITY_COLUMNS.map((column) => (
        <span key={column.key} className={`text-right tabular-nums ${className}`}>
          {formatCompactQuantity(quantities[column.key])}
        </span>
      ))}
    </>
  );
}

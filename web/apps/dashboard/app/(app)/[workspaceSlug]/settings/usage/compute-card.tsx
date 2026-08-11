"use client";

// The Gateway row's Keys and Cost cells are [no data] placeholders: ClickHouse
// counts distinct verified keys per workspace only, so no per-project figure exists.

import { formatCompactQuantity, formatDollars } from "@/lib/fmt";
import { ChevronRight, Cube } from "@unkey/icons";
import {
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemGroup,
  ItemHeader,
  ItemMedia,
  ItemSeparator,
  ItemTitle,
  Skeleton,
} from "@unkey/ui";
import { Fragment, type ReactNode, useState } from "react";
import {
  type ComputeTree,
  type UsageApp,
  type UsageProject,
  type UsageQuantities,
  microCentsToDisplayCents,
} from "./compute-tree";

const ROW = "flex items-center gap-3 px-4";

const BAND =
  "border-grayA-4 border-y bg-grayA-2 py-2 font-semibold text-[10px] text-gray-9 uppercase tracking-wider";

const AMOUNT = "font-semibold text-2xl text-gray-12 leading-tight tracking-tight tabular-nums";

const ROW_ICON = "size-5 border border-grayA-4 bg-gray-1";

const COST = "w-20";

const KEYS = "w-24";

const QUANTITY_COLUMNS: ReadonlyArray<{
  key: keyof UsageQuantities;
  label: string;
  width: string;
}> = [
  { key: "cpuHours", label: "CPU hrs", width: "w-16" },
  { key: "memoryGiBHours", label: "Memory GiB-hrs", width: "w-24" },
  { key: "egressGiB", label: "Egress GiB", width: "w-20" },
  { key: "diskGiBHours", label: "Disk GiB-hrs", width: "w-24" },
];

const SKELETON_ROWS = ["first", "second", "third"];

export function ComputeCardShell({
  description,
  amount,
  children,
}: {
  description: string;
  amount?: ReactNode;
  children: ReactNode;
}) {
  return (
    <ItemGroup variant="outline">
      <ItemHeader>
        <ItemMedia className="bg-orangeA-3 text-orange-11">
          <Cube />
        </ItemMedia>
        <ItemContent>
          <ItemTitle>Compute</ItemTitle>
          <ItemDescription>{description}</ItemDescription>
        </ItemContent>
        {amount === undefined ? null : <ItemActions className={AMOUNT}>{amount}</ItemActions>}
      </ItemHeader>
      <ItemSeparator />
      {children}
    </ItemGroup>
  );
}

export function ComputeCardSkeleton() {
  return (
    <ComputeCardShell
      description="Usage per project this period"
      amount={<Skeleton className="h-6 w-20" />}
    >
      {SKELETON_ROWS.map((row, index) => (
        <Fragment key={row}>
          {index === 0 ? null : <ItemSeparator />}
          <Item className="gap-2">
            <ChevronRight iconSize="sm-regular" className="shrink-0 text-gray-6" />
            <ItemMedia className={ROW_ICON}>
              <Cube />
            </ItemMedia>
            <ItemContent>
              <Skeleton className="h-3 w-40" />
            </ItemContent>
            <ItemActions className={`${COST} justify-end`}>
              <Skeleton className="h-3 w-12" />
            </ItemActions>
          </Item>
        </Fragment>
      ))}
    </ComputeCardShell>
  );
}

type ComputeCardProps = {
  tree: ComputeTree;
  /**
   * Workspace gross for the period. It covers the verified keys the tree has no
   * grain for, so it is read from the workspace endpoint rather than summed here.
   * Null when that endpoint failed.
   */
  workspaceCents: number | null;
};

export function ComputeCard({ tree, workspaceCents }: ComputeCardProps) {
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
    <ComputeCardShell
      description="Usage per project this period"
      amount={
        workspaceCents === null ? (
          <span className="font-normal text-[13px] text-gray-9">Unavailable</span>
        ) : (
          formatDollars(workspaceCents)
        )
      }
    >
      {tree.projects.length === 0 ? (
        <Item>
          <ItemContent>
            <ItemDescription>No compute usage recorded this period.</ItemDescription>
          </ItemContent>
        </Item>
      ) : (
        tree.projects.map((project, index) => (
          <Fragment key={project.projectId}>
            {index === 0 ? null : <ItemSeparator className="bg-gray-5" />}
            <ProjectRow
              project={project}
              open={open.has(project.projectId)}
              onToggle={() => toggle(project.projectId)}
            />
          </Fragment>
        ))
      )}
    </ComputeCardShell>
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
      <Item
        className="gap-2"
        render={<button type="button" aria-expanded={open} onClick={onToggle} />}
      >
        <ChevronRight
          iconSize="sm-regular"
          className={`shrink-0 text-gray-9 transition-transform duration-150 ease-out motion-reduce:transition-none ${open ? "rotate-90" : ""}`}
        />
        <ItemMedia className={ROW_ICON}>
          <Cube />
        </ItemMedia>
        <ItemContent>
          <ItemTitle className="truncate">{project.name}</ItemTitle>
        </ItemContent>
        <ItemActions className={`${COST} justify-end font-medium tabular-nums`}>
          {formatDollars(microCentsToDisplayCents(project.microCents))}
        </ItemActions>
      </Item>
      <div
        className="grid transition-[grid-template-rows] duration-200 ease-out motion-reduce:transition-none"
        style={{ gridTemplateRows: open ? "1fr" : "0fr" }}
      >
        <div className="overflow-hidden">
          <div className={`${ROW} ${BAND}`}>
            <div className="min-w-0 flex-1">App</div>
            {QUANTITY_COLUMNS.map((column) => (
              <div key={column.key} className={`${column.width} text-right`}>
                {column.label}
              </div>
            ))}
            <div className={`${COST} text-right`}>Cost</div>
          </div>
          {project.apps.map((app) => (
            <AppRows key={app.appId} app={app} />
          ))}
          <div className={`${ROW} ${BAND}`}>
            <div className="min-w-0 flex-1">Gateway</div>
            <div className={`${KEYS} text-right`}>Keys</div>
            <div className={`${COST} text-right`}>Cost</div>
          </div>
          <div className={`${ROW} py-2.5`}>
            <span className="min-w-0 flex-1 truncate font-medium text-[13px] text-gray-12">
              Verified keys
            </span>
            <span className={`${KEYS} text-right text-[13px] text-gray-9`}>[no data]</span>
            <span className={`${COST} text-right text-[13px] text-gray-9`}>[no data]</span>
          </div>
        </div>
      </div>
    </div>
  );
}

function AppRows({ app }: { app: UsageApp }) {
  return (
    <div>
      <div className={`${ROW} pt-2.5 pb-1`}>
        <span className="min-w-0 flex-1 truncate font-medium text-[13px] text-gray-12">
          {app.name}
        </span>
        <Quantities quantities={app} className="text-[13px] text-gray-11" />
        <span className={`${COST} text-right font-medium text-[13px] text-gray-12 tabular-nums`}>
          {formatDollars(microCentsToDisplayCents(app.microCents))}
        </span>
      </div>
      {app.environments.map((environment) => (
        <div key={environment.environmentId} className={`${ROW} py-1 last:pb-2.5`}>
          <span className="min-w-0 flex-1 truncate text-gray-10 text-xs">{environment.name}</span>
          <Quantities quantities={environment} className="text-gray-10 text-xs" />
          <span className={`${COST} text-right text-gray-11 text-xs tabular-nums`}>
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
        <span key={column.key} className={`${column.width} text-right tabular-nums ${className}`}>
          {formatCompactQuantity(quantities[column.key])}
        </span>
      ))}
    </>
  );
}

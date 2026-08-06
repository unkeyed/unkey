"use client";

/**
 * The live coverage panel: given a valid resource-path pattern, show the total
 * blast radius, then matches grouped by resource type with a per-type coverage
 * meter. Concrete (non-pattern) paths get a distinct "exactly one resource"
 * card instead of group meters.
 */

import { cn } from "@unkey/ui";
import { useMemo, useState } from "react";
import { RESOURCE_TYPES } from "../lib/catalog";
import { ALL_RESOURCES, type ConcreteResource, coverage } from "../lib/mock-data";
import { isPattern } from "../lib/urn";

const TYPE_META = new Map(
  RESOURCE_TYPES.map((def, index) => [def.type, { label: def.label, order: index }]),
);

const TYPE_TOTALS: ReadonlyMap<string, number> = (() => {
  const totals = new Map<string, number>();
  for (const resource of ALL_RESOURCES) {
    totals.set(resource.type, (totals.get(resource.type) ?? 0) + 1);
  }
  return totals;
})();

const MAX_VISIBLE_ROWS = 8;

export function ResourceRow({
  resource,
  tone,
}: {
  resource: ConcreteResource;
  tone?: "gained" | "lost";
}) {
  return (
    <div
      className={cn(
        "flex items-center justify-between gap-4 rounded-md px-2 py-1.5",
        tone === "gained" && "bg-successA-3",
        tone === "lost" && "bg-errorA-2",
        tone === undefined && "hover:bg-grayA-2 transition-colors",
      )}
    >
      <span className="flex items-center gap-1.5 min-w-0">
        {tone === "gained" && (
          <span aria-hidden="true" className="font-mono text-xs text-success-11 shrink-0">
            +
          </span>
        )}
        {tone === "lost" && (
          <span aria-hidden="true" className="font-mono text-xs text-error-11 shrink-0">
            &minus;
          </span>
        )}
        <span
          className={cn(
            "text-[13px] truncate",
            tone === "gained" && "text-success-11",
            tone === "lost" && "text-error-11",
            tone === undefined && "text-gray-12",
          )}
          title={resource.label}
        >
          {resource.label}
        </span>
      </span>
      <span className="font-mono text-xs text-gray-9 truncate min-w-0" title={resource.path}>
        {resource.path}
      </span>
    </div>
  );
}

export function ExpanderButton({
  hidden,
  expanded,
  onToggle,
}: {
  hidden: number;
  expanded: boolean;
  onToggle: () => void;
}) {
  if (hidden === 0 && !expanded) {
    return null;
  }
  return (
    <button
      type="button"
      onClick={onToggle}
      className="self-start px-2 text-xs text-gray-10 hover:text-gray-12 transition-colors"
    >
      {expanded ? "Show fewer" : `+${hidden} more`}
    </button>
  );
}

function ResourceGroup({ type, resources }: { type: string; resources: ConcreteResource[] }) {
  const [expanded, setExpanded] = useState(false);
  const meta = TYPE_META.get(type);
  const total = TYPE_TOTALS.get(type) ?? resources.length;
  const percent = total === 0 ? 0 : Math.round((resources.length / total) * 100);
  const visible = expanded ? resources : resources.slice(0, MAX_VISIBLE_ROWS);
  const hidden = resources.length - visible.length;
  const overflows = resources.length > MAX_VISIBLE_ROWS;

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-baseline justify-between gap-4">
        <span className="text-[13px] font-medium text-gray-12">{meta?.label ?? type}</span>
        <span className="text-xs text-gray-10 tabular-nums">
          {resources.length} of {total}
        </span>
      </div>
      <div className="h-1 w-full rounded-full bg-grayA-3 overflow-hidden">
        <div
          className="h-full rounded-full bg-accent-9"
          style={{ width: `${Math.max(percent, 2)}%` }}
        />
      </div>
      <div className="flex flex-col">
        {visible.map((resource) => (
          <ResourceRow key={resource.path} resource={resource} />
        ))}
      </div>
      {overflows && (
        <ExpanderButton
          hidden={hidden}
          expanded={expanded}
          onToggle={() => setExpanded((prev) => !prev)}
        />
      )}
    </div>
  );
}

function ExactPathCard({ path, match }: { path: string; match: ConcreteResource | undefined }) {
  return (
    <div className="rounded-lg border border-grayA-4 p-5 flex flex-col gap-3">
      <div className="flex items-center gap-2 flex-wrap">
        <span className="rounded bg-grayA-3 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-gray-11">
          Exact path
        </span>
        <span className="text-xs text-gray-10">
          No wildcards. This grant reaches exactly one resource.
        </span>
      </div>
      {match ? (
        <div className="flex flex-col gap-1 min-w-0">
          <span className="text-2xl font-semibold text-gray-12 truncate" title={match.label}>
            {match.label}
          </span>
          <span className="font-mono text-xs text-gray-9 truncate" title={match.path}>
            {match.path}
          </span>
          <span className="text-xs text-gray-10">
            {TYPE_META.get(match.type)?.label ?? match.type}
          </span>
        </div>
      ) : (
        <p className="text-sm text-gray-11">
          Nothing exists at <span className="font-mono text-xs text-gray-12">{path}</span> in the
          ACME workspace today. The grant is still valid and applies the moment a resource is
          created at this exact path.
        </p>
      )}
    </div>
  );
}

export function CoveragePanel({ pattern, className }: { pattern: string; className?: string }) {
  const matches = useMemo(() => coverage(pattern), [pattern]);
  const concrete = !isPattern(pattern);

  const groups = useMemo(() => {
    const byType = new Map<string, ConcreteResource[]>();
    for (const match of matches) {
      const list = byType.get(match.type);
      if (list) {
        list.push(match);
      } else {
        byType.set(match.type, [match]);
      }
    }
    return [...byType.entries()].sort(
      (a, b) =>
        (TYPE_META.get(a[0])?.order ?? Number.MAX_SAFE_INTEGER) -
        (TYPE_META.get(b[0])?.order ?? Number.MAX_SAFE_INTEGER),
    );
  }, [matches]);

  if (concrete) {
    return (
      <div className={className}>
        <ExactPathCard path={pattern} match={matches[0]} />
      </div>
    );
  }

  return (
    <div className={cn("flex flex-col gap-6", className)}>
      <div className="flex flex-col gap-0.5">
        <div className="flex items-baseline gap-2">
          <span className="text-3xl font-semibold text-gray-12 tabular-nums">{matches.length}</span>
          <span className="text-lg text-gray-11">
            {matches.length === 1 ? "resource" : "resources"}
          </span>
        </div>
        <span className="text-xs text-gray-10">
          covered by this pattern, out of {ALL_RESOURCES.length} in the ACME workspace
        </span>
      </div>
      {matches.length === 0 ? (
        <div className="rounded-lg border border-grayA-4 bg-grayA-2 p-5">
          <p className="text-sm text-gray-11">
            No resources match this pattern. Check the collection names in each segment, or widen
            the scope with <span className="font-mono text-xs text-gray-12">*</span> for one segment
            or a trailing <span className="font-mono text-xs text-gray-12">**</span> for everything
            below a path.
          </p>
        </div>
      ) : (
        <div className="flex flex-col gap-7">
          {groups.map(([type, resources]) => (
            <ResourceGroup key={type} type={type} resources={resources} />
          ))}
        </div>
      )}
    </div>
  );
}

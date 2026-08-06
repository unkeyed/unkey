"use client";

/**
 * Permission chip wrapped in a blast-radius hover card: hovering any grant
 * shows how many concrete resources it covers today, a preview of the first
 * few, and where the grant comes from (direct or via a role).
 */

import { HoverCard, HoverCardContent, HoverCardTrigger, cn } from "@unkey/ui";
import { PermissionChip } from "../components/urn-display";
import { type ConcreteResource, coverage } from "../lib/mock-data";
import { parsePermission } from "../lib/urn";

const PREVIEW_LIMIT = 5;

function CoveragePreview({ covered }: { covered: ConcreteResource[] }) {
  if (covered.length === 0) {
    return (
      <p className="text-xs text-gray-10">
        This pattern matches nothing in the workspace yet. It will apply to resources created later.
      </p>
    );
  }
  const preview = covered.slice(0, PREVIEW_LIMIT);
  const remaining = covered.length - preview.length;
  return (
    <div className="flex flex-col gap-1">
      {preview.map((resource) => (
        <div key={resource.path} className="flex items-baseline justify-between gap-3 min-w-0">
          <span className="text-xs text-gray-12 truncate">{resource.label}</span>
          <span className="font-mono text-[10px] text-gray-9 truncate shrink-0 max-w-[55%]">
            {resource.path}
          </span>
        </div>
      ))}
      {remaining > 0 && <span className="text-xs text-gray-9">+{remaining} more</span>}
    </div>
  );
}

export function GrantChip({
  value,
  source,
  onRemove,
  dimmed,
}: {
  /** canonical permission string */
  value: string;
  /** e.g. "Direct grant" or "Role key-minter" */
  source: string;
  onRemove?: () => void;
  dimmed?: boolean;
}) {
  const parsed = parsePermission(value);
  const covered = parsed.ok ? coverage(parsed.value.urn.resource) : [];

  return (
    <HoverCard>
      <HoverCardTrigger
        delay={150}
        closeDelay={100}
        render={
          <span className={cn("inline-flex max-w-full", dimmed && "opacity-70")}>
            <PermissionChip value={value} onRemove={onRemove} />
          </span>
        }
      />
      <HoverCardContent align="start" className="w-80 p-3">
        <div className="flex flex-col gap-2.5">
          <span className="text-xs font-medium text-gray-12">
            {covered.length === 0
              ? "Covers no resources today"
              : covered.length === 1
                ? "Covers 1 resource today"
                : `Covers ${covered.length} resources today`}
          </span>
          <CoveragePreview covered={covered} />
          <div className="border-t border-grayA-4 pt-2">
            <span className="text-[11px] uppercase tracking-wide text-gray-9">Source</span>
            <p className="text-xs text-gray-11">{source}</p>
          </div>
        </div>
      </HoverCardContent>
    </HoverCard>
  );
}

export function LegacyGrantChip({ value }: { value: string }) {
  return (
    <HoverCard>
      <HoverCardTrigger
        delay={150}
        closeDelay={100}
        render={
          <span className="inline-flex max-w-full">
            <PermissionChip value={value} legacy />
          </span>
        }
      />
      <HoverCardContent align="start" className="w-72 p-3">
        <div className="flex flex-col gap-1.5">
          <span className="text-xs font-medium text-gray-12">Legacy permission</span>
          <p className="text-xs text-gray-11">
            Pre-URN tuple permission. It matches this exact string only and has no wildcard
            coverage. Migrate it to a URN grant to see its blast radius.
          </p>
        </div>
      </HoverCardContent>
    </HoverCard>
  );
}

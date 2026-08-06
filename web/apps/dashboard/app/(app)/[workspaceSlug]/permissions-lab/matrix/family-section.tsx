"use client";

import { Check, ChevronDown, ChevronRight, Plus } from "@unkey/icons";
import { Button, InfoTooltip, cn } from "@unkey/ui";
import { UrnText } from "../components/urn-display";
import type { ActionDef } from "../lib/catalog";
import { type ConcreteResource, labelForPath } from "../lib/mock-data";
import {
  type Family,
  type ParsedGrant,
  type PatternRow,
  cellState,
  sourceLabel,
} from "./matrix-model";

export interface FamilySectionProps {
  family: Family;
  /** rows after the search filter */
  rows: ConcreteResource[];
  patternRows: PatternRow[];
  grants: ParsedGrant[];
  expanded: boolean;
  /** searching pins every section open, so the toggle is disabled */
  searching: boolean;
  onToggle: () => void;
  onGrant: (resource: ConcreteResource, action: string) => void;
  onRequestRevoke: (
    anchor: HTMLElement,
    resource: ConcreteResource,
    action: string,
    grant: ParsedGrant,
  ) => void;
  onRequestMaterialize: (anchor: HTMLElement, row: PatternRow) => void;
}

export function FamilySection({
  family,
  rows,
  patternRows,
  grants,
  expanded,
  searching,
  onToggle,
  onGrant,
  onRequestRevoke,
  onRequestMaterialize,
}: FamilySectionProps) {
  const actions = family.def.actions;
  // resource column + one per action + trailing controls column
  const columnCount = actions.length + 2;
  const groups = groupRows(family, rows);

  return (
    <section className="rounded-lg border border-grayA-4 overflow-hidden">
      <button
        type="button"
        onClick={onToggle}
        disabled={searching}
        aria-expanded={expanded}
        className="flex w-full items-center gap-2 bg-grayA-2 px-3 py-2.5 text-left disabled:cursor-default"
      >
        <span className="text-gray-9">
          {expanded ? (
            <ChevronDown iconSize="sm-regular" />
          ) : (
            <ChevronRight iconSize="sm-regular" />
          )}
        </span>
        <span className="text-[13px] font-medium text-gray-12">{family.title}</span>
        <span className="text-xs text-gray-10">
          {searching ? `${rows.length} of ${family.resources.length}` : rows.length}{" "}
          {rows.length === 1 ? "resource" : "resources"}
        </span>
        {patternRows.length > 0 && (
          <span className="rounded bg-warningA-3 px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-warning-11">
            {patternRows.length} pattern {patternRows.length === 1 ? "grant" : "grants"}
          </span>
        )}
      </button>

      {expanded && (
        <div className="overflow-x-auto border-t border-grayA-4">
          <table className="w-full min-w-max border-collapse text-[13px]">
            <thead>
              <tr className="border-b border-grayA-4">
                <th className="w-[340px] min-w-[260px] px-3 py-2 text-left text-xs font-normal text-gray-10">
                  Resource
                </th>
                {actions.map((action) => (
                  <ActionHeader key={action.action} action={action} />
                ))}
                <th className="w-[130px] px-3 py-2" aria-label="Row controls" />
              </tr>
            </thead>
            <tbody>
              {patternRows.map((row) => (
                <PatternGrantRow
                  key={row.grant.raw}
                  family={family}
                  row={row}
                  onRequestMaterialize={onRequestMaterialize}
                />
              ))}
              {rows.length === 0 && (
                <tr>
                  <td colSpan={columnCount} className="px-3 py-6 text-center text-xs text-gray-10">
                    No {family.title.toLowerCase()} match the current search.
                  </td>
                </tr>
              )}
              {groups.map((group) => (
                <GroupRows
                  key={group.parentPath ?? "__ungrouped"}
                  group={group}
                  family={family}
                  grants={grants}
                  columnCount={columnCount}
                  onGrant={onGrant}
                  onRequestRevoke={onRequestRevoke}
                />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

// ---------------------------------------------------------------------------
// Header
// ---------------------------------------------------------------------------

function ActionHeader({ action }: { action: ActionDef }) {
  return (
    <th className="px-2 py-2 text-center font-normal">
      <InfoTooltip
        asChild
        position={{ side: "top" }}
        content={
          <div className="flex max-w-[240px] flex-col gap-0.5 text-left">
            <span className="font-mono text-xs text-gray-12">{action.action}</span>
            <span className="font-normal text-gray-11">{action.description}</span>
            {!action.implemented && (
              <span className="font-normal text-gray-9">Planned action, not implemented yet.</span>
            )}
          </div>
        }
      >
        <span
          className={cn(
            "cursor-help font-mono text-xs",
            action.implemented ? "text-gray-11" : "text-gray-9",
          )}
        >
          {action.action}
        </span>
      </InfoTooltip>
    </th>
  );
}

// ---------------------------------------------------------------------------
// Pattern rows
// ---------------------------------------------------------------------------

function PatternGrantRow({
  family,
  row,
  onRequestMaterialize,
}: {
  family: Family;
  row: PatternRow;
  onRequestMaterialize: (anchor: HTMLElement, row: PatternRow) => void;
}) {
  const roleSources = row.grant.sources.filter((s) => s.kind === "role");
  const isDirect = row.grant.sources.some((s) => s.kind === "direct");

  return (
    <tr className="border-b border-warningA-4 bg-warningA-2">
      <td className="px-3 py-2">
        <div className="flex min-w-0 flex-col gap-0.5">
          <UrnText value={row.grant.raw} />
          <span className="text-[11px] text-gray-10">
            Pattern grant covering {row.covered.length} of {family.resources.length}{" "}
            {family.title.toLowerCase()}
            {roleSources.length > 0 &&
              ` (${roleSources.map((s) => `role: ${s.roleName}`).join(", ")})`}
          </span>
        </div>
      </td>
      {family.def.actions.map((action) => (
        <td key={action.action} className="px-2 py-2 text-center">
          {row.litActions.has(action.action) && (
            <span
              className="inline-flex size-7 items-center justify-center text-warning-11"
              aria-label={`${action.action} covered by pattern`}
            >
              <Check iconSize="sm-regular" />
            </span>
          )}
        </td>
      ))}
      <td className="px-3 py-2 text-right">
        {row.materializable ? (
          <Button
            variant="outline"
            size="sm"
            disabled={row.covered.length === 0}
            onClick={(e) => onRequestMaterialize(e.currentTarget, row)}
          >
            Materialize
          </Button>
        ) : (
          <InfoTooltip
            asChild
            position={{ side: "left" }}
            content={
              <span className="block max-w-[220px] font-normal text-gray-11">
                {isDirect
                  ? "A wildcard action spans every resource family, so it cannot be materialized from one section."
                  : "This grant comes from a role. Edit the role to change it."}
              </span>
            }
          >
            <span className="cursor-help text-[11px] text-gray-10">
              {isDirect ? "not materializable" : "via role"}
            </span>
          </InfoTooltip>
        )}
      </td>
    </tr>
  );
}

// ---------------------------------------------------------------------------
// Resource rows
// ---------------------------------------------------------------------------

interface RowGroup {
  /** set when the family nests under a parent (keys under their keyspace) */
  parentPath: string | null;
  rows: ConcreteResource[];
}

/** Keys get subgroup header rows per keyspace; other families stay flat. */
function groupRows(family: Family, rows: ConcreteResource[]): RowGroup[] {
  if (family.type !== "key") {
    return rows.length > 0 ? [{ parentPath: null, rows }] : [];
  }
  const groups = new Map<string, ConcreteResource[]>();
  for (const row of rows) {
    const parentPath = row.path.split("/").slice(0, 2).join("/");
    const bucket = groups.get(parentPath);
    if (bucket) {
      bucket.push(row);
    } else {
      groups.set(parentPath, [row]);
    }
  }
  return [...groups.entries()].map(([parentPath, groupedRows]) => ({
    parentPath,
    rows: groupedRows,
  }));
}

function GroupRows({
  group,
  family,
  grants,
  columnCount,
  onGrant,
  onRequestRevoke,
}: {
  group: RowGroup;
  family: Family;
  grants: ParsedGrant[];
  columnCount: number;
  onGrant: (resource: ConcreteResource, action: string) => void;
  onRequestRevoke: (
    anchor: HTMLElement,
    resource: ConcreteResource,
    action: string,
    grant: ParsedGrant,
  ) => void;
}) {
  return (
    <>
      {group.parentPath !== null && (
        <tr className="border-t border-grayA-3 bg-grayA-2">
          <td colSpan={columnCount} className="px-3 py-1.5 text-[11px] text-gray-10">
            <span className="font-medium text-gray-11">{labelForPath(group.parentPath)}</span>{" "}
            <span className="font-mono">{group.parentPath}</span> &middot; {group.rows.length}{" "}
            {group.rows.length === 1 ? "key" : "keys"}
          </td>
        </tr>
      )}
      {group.rows.map((resource) => (
        <tr key={resource.path} className="group/row border-t border-grayA-3 hover:bg-grayA-2">
          <td className="px-3 py-1">
            <div className="flex min-w-0 items-baseline gap-2">
              <span className="truncate text-gray-12">{resource.label}</span>
              <span className="truncate font-mono text-[11px] text-gray-9">{resource.path}</span>
            </div>
          </td>
          {family.def.actions.map((action) => (
            <MatrixCell
              key={action.action}
              resource={resource}
              action={action.action}
              grants={grants}
              onGrant={onGrant}
              onRequestRevoke={onRequestRevoke}
            />
          ))}
          <td className="px-3 py-1" />
        </tr>
      ))}
    </>
  );
}

function MatrixCell({
  resource,
  action,
  grants,
  onGrant,
  onRequestRevoke,
}: {
  resource: ConcreteResource;
  action: string;
  grants: ParsedGrant[];
  onGrant: (resource: ConcreteResource, action: string) => void;
  onRequestRevoke: (
    anchor: HTMLElement,
    resource: ConcreteResource,
    action: string,
    grant: ParsedGrant,
  ) => void;
}) {
  const state = cellState(grants, resource.path, action);

  if (state.kind === "direct") {
    return (
      <td className="px-2 py-1 text-center">
        <InfoTooltip
          asChild
          position={{ side: "top" }}
          content={
            <span className="font-normal text-gray-11">Granted directly. Click to revoke.</span>
          }
        >
          <button
            type="button"
            onClick={(e) => onRequestRevoke(e.currentTarget, resource, action, state.grant)}
            aria-label={`Revoke ${action} on ${resource.label}`}
            className="inline-flex size-7 items-center justify-center rounded-md bg-successA-3 text-success-11 transition-colors hover:bg-errorA-3 hover:text-error-11"
          >
            <Check iconSize="sm-regular" />
          </button>
        </InfoTooltip>
      </td>
    );
  }

  if (state.kind === "covered") {
    return (
      <td className="px-2 py-1 text-center">
        <InfoTooltip
          asChild
          position={{ side: "top" }}
          content={
            <div className="flex max-w-[320px] flex-col gap-1 text-left">
              {state.grants.map((grant) => (
                <span key={grant.raw} className="font-mono text-xs text-gray-12">
                  {sourceLabel(grant)}
                </span>
              ))}
              <span className="font-normal text-gray-10">
                Inherited access cannot be revoked from this grid.
              </span>
            </div>
          }
        >
          <span
            className="inline-flex size-7 cursor-help items-center justify-center rounded-md border border-successA-5 text-success-9"
            aria-label={`${action} on ${resource.label} inherited`}
          >
            <Check iconSize="sm-regular" />
          </span>
        </InfoTooltip>
      </td>
    );
  }

  return (
    <td className="px-2 py-1 text-center">
      <button
        type="button"
        onClick={() => onGrant(resource, action)}
        aria-label={`Grant ${action} on ${resource.label}`}
        className="inline-flex size-7 items-center justify-center rounded-md text-gray-9 opacity-0 transition-opacity hover:bg-grayA-3 hover:text-gray-12 focus-visible:opacity-100 group-hover/row:opacity-100"
      >
        <Plus iconSize="sm-regular" />
      </button>
    </td>
  );
}

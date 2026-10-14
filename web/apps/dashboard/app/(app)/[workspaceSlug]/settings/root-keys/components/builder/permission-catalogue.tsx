"use client";

import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { IconChevronRightOutline18 } from "@unkey/icons";
import { Checkbox, Input } from "@unkey/ui";
import { useId, useState } from "react";
import { catalogueRows } from "./lib/catalogue";
import {
  ACTIONS,
  type Action,
  type CatalogueGroup,
  type PermissionRow,
  type PermissionSelection,
  type ScopeCatalogue,
  offeredActions,
  rowOffers,
} from "./lib/catalogue.types";
import {
  countSelectedActions,
  rowActions,
  setRowsActions,
  toggleRowAction,
  toggleRowsAction,
} from "./lib/policy";
import { PermissionCatalogueBulkMenu } from "./permission-catalogue-bulk-menu";
import { ACTION_LABELS, PermissionCatalogueRow } from "./permission-catalogue-row";

type PermissionCatalogueProps = {
  catalogue: ScopeCatalogue;
  value: PermissionSelection;
  onChange: (selection: PermissionSelection) => void;
};

function matches(group: CatalogueGroup, query: string): PermissionRow[] {
  return group.rows.filter((row) => query.length === 0 || row.label.toLowerCase().includes(query));
}

export function PermissionCatalogue({ catalogue, value, onChange }: PermissionCatalogueProps) {
  const [search, setSearch] = useState("");
  const [closedGroups, setClosedGroups] = useState<string[]>([]);
  const query = search.trim().toLowerCase();

  const matchedGroups = catalogue.groups.map((group) => ({ group, rows: matches(group, query) }));
  const visible = new Set(matchedGroups.flatMap(({ rows }) => rows.map((row) => row.id)));
  const hidden = catalogueRows(catalogue).filter(
    (row) => !visible.has(row.id) && rowActions(value, row.id).length > 0,
  ).length;

  const setActions = (actions: readonly Action[]) => {
    onChange(setRowsActions(value, catalogueRows(catalogue), actions));
  };

  const toggleGroup = (groupId: string, open: boolean) => {
    setClosedGroups((current) =>
      open ? current.filter((entry) => entry !== groupId) : [...current, groupId],
    );
  };

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-2">
        <Input
          value={search}
          onChange={(event) => setSearch(event.currentTarget.value)}
          placeholder="Search permissions…"
          className="h-8"
        />
        <PermissionCatalogueBulkMenu onSelect={setActions} />
      </div>

      {hidden > 0 ? (
        <span className="text-xs text-warning-11">
          {hidden === 1
            ? "1 selected row hidden by this filter"
            : `${hidden} selected rows hidden by this filter`}
        </span>
      ) : null}

      <div className="flex flex-col divide-y divide-grayA-3">
        {matchedGroups.map(({ group, rows }) => {
          if (rows.length === 0) {
            return null;
          }
          const selected = countSelectedActions(value, group.rows);
          const total = group.rows.reduce((sum, row) => sum + offeredActions(row).length, 0);
          const columns = ACTIONS.filter((action) =>
            group.rows.some((row) => rowOffers(row, action)),
          );

          return (
            <Collapsible
              key={group.id}
              open={query.length > 0 || !closedGroups.includes(group.id)}
              onOpenChange={(open) => toggleGroup(group.id, open)}
            >
              <div className="flex flex-wrap items-center justify-between gap-x-4">
                <CollapsibleTrigger className="flex flex-1 min-w-40 items-center gap-3 py-2.5 [&[data-panel-open]>svg]:rotate-90">
                  <IconChevronRightOutline18
                    className="size-3 shrink-0 transition-transform duration-200 text-gray-11"
                    aria-hidden="true"
                  />
                  <span className="text-sm text-gray-12">{group.label}</span>
                  <span className="ml-auto text-xs text-gray-9 tabular-nums">
                    {selected}/{total}
                  </span>
                </CollapsibleTrigger>
                <PermissionGroupActions
                  group={group}
                  columns={columns}
                  value={value}
                  onChange={onChange}
                />
              </div>
              <CollapsibleContent>
                <div className="flex flex-col pl-6 pb-2">
                  {rows.map((row) => (
                    <PermissionCatalogueRow
                      key={row.id}
                      row={row}
                      actions={rowActions(value, row.id)}
                      columns={columns}
                      onToggle={(action, next) =>
                        onChange(toggleRowAction(value, row.id, action, next))
                      }
                    />
                  ))}
                </div>
              </CollapsibleContent>
            </Collapsible>
          );
        })}
      </div>

      {query.length > 0 && visible.size === 0 ? (
        <span className="text-xs text-gray-10">No permissions match “{search.trim()}”.</span>
      ) : null}
    </div>
  );
}

function PermissionGroupActions({
  group,
  columns,
  value,
  onChange,
}: {
  group: CatalogueGroup;
  columns: readonly Action[];
  value: PermissionSelection;
  onChange: (selection: PermissionSelection) => void;
}) {
  const id = useId();
  return (
    <div className="flex items-center gap-4 shrink-0 ml-auto py-2.5">
      {columns.map((action) => {
        const eligible = group.rows.filter((row) => rowOffers(row, action));
        const picked = eligible.filter((row) => rowActions(value, row.id).includes(action));
        const checkboxId = `${id}-${action}`;
        return (
          <div key={action} className="flex items-center gap-2">
            <Checkbox
              id={checkboxId}
              size="md"
              aria-label={`${ACTION_LABELS[action]} all permissions in ${group.label}`}
              checked={
                picked.length === eligible.length
                  ? true
                  : picked.length > 0
                    ? "indeterminate"
                    : false
              }
              onCheckedChange={(next) =>
                onChange(toggleRowsAction(value, group.rows, action, next === true))
              }
            />
            <label htmlFor={checkboxId} className="cursor-pointer select-none text-xs text-gray-12">
              {ACTION_LABELS[action]}
            </label>
          </div>
        );
      })}
    </div>
  );
}

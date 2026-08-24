"use client";

import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { ChevronRight } from "@unkey/icons";
import { Input } from "@unkey/ui";
import { useState } from "react";
import { catalogueRows } from "../lib/catalogue";
import {
  ACTIONS,
  type Action,
  type CatalogueGroup,
  type PermissionSelection,
  type ScopeCatalogue,
} from "../lib/catalogue.types";
import { countSelectedActions, rowActions, setRowsActions, toggleRowAction } from "../lib/policy";
import { PermissionCatalogueBulkMenu } from "./permission-catalogue-bulk-menu";
import { PermissionCatalogueRow } from "./permission-catalogue-row";

type PermissionCatalogueProps = {
  catalogue: ScopeCatalogue;
  value: PermissionSelection;
  onChange: (selection: PermissionSelection) => void;
};

const matches = (group: CatalogueGroup, query: string) =>
  group.rows.filter((row) => query.length === 0 || row.label.toLowerCase().includes(query));

export function PermissionCatalogue({ catalogue, value, onChange }: PermissionCatalogueProps) {
  const [search, setSearch] = useState("");
  const [closedGroups, setClosedGroups] = useState<string[]>([]);
  const query = search.trim().toLowerCase();

  const visible = new Set(
    catalogue.groups.flatMap((group) => matches(group, query).map((row) => row.id)),
  );
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
        {catalogue.groups.map((group) => {
          const rows = matches(group, query);
          if (rows.length === 0) {
            return null;
          }
          const selected = countSelectedActions(value, group.rows);
          const total = group.rows.length * ACTIONS.length;

          return (
            <Collapsible
              key={group.id}
              open={query.length > 0 || !closedGroups.includes(group.id)}
              onOpenChange={(open) => toggleGroup(group.id, open)}
            >
              <CollapsibleTrigger className="flex items-center gap-3 w-full py-2.5 [&[data-panel-open]>svg]:rotate-90">
                <ChevronRight
                  className="size-3 transition-transform duration-200 text-gray-11"
                  aria-hidden="true"
                />
                <span className="text-[13px] text-accent-12">{group.label}</span>
                <span className="ml-auto text-xs text-gray-9 tabular-nums">
                  {selected}/{total}
                </span>
              </CollapsibleTrigger>
              <CollapsibleContent>
                <div className="flex flex-col pl-7 pb-2">
                  {rows.map((row) => (
                    <PermissionCatalogueRow
                      key={row.id}
                      row={row}
                      actions={rowActions(value, row.id)}
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

"use client";

import { Table } from "@tanstack/react-table";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { DataTableViewOptions } from "./table-view-options";

import { X } from "lucide-react";
import { DataTableFacetedFilter } from "./table-faceted-filter";

interface DataTableToolbarProps<TData> {
  table: Table<TData>;
}

export const events = [
  "workspace.create",
  "workspace.update",
  "workspace.delete",
  "api.create",
  "api.update",
  "api.delete",
  "key.create",
  "key.update",
  "key.delete",
  "vercelIntegration.create",
  "vercelIntegration.update",
  "vercelIntegration.delete",
  "vercelBinding.create",
  "vercelBinding.update",
  "vercelBinding.delete",
  "role.create",
  "role.update",
  "role.delete",
  "permission.create",
  "permission.update",
  "permission.delete",
  "authorization.connect_role_and_permission",
  "authorization.disconnect_role_and_permissions",
  "authorization.connect_role_and_key",
  "authorization.disconnect_role_and_key",
  "authorization.connect_permission_and_key",
  "authorization.disconnect_permission_and_key",
];

export function DataTableToolbar<TData>({ table }: DataTableToolbarProps<TData>) {
  const isFiltered = table.getState().columnFilters.length > 0;

  return (
    <div className="flex items-center justify-between">
      <div className="flex items-center flex-1 space-x-2">
        <Input
          placeholder="Filter events..."
          value={(table.getColumn("description")?.getFilterValue() as string) ?? ""}
          onChange={(event) => table.getColumn("description")?.setFilterValue(event.target.value)}
          className="h-8 w-[150px] lg:w-[250px]"
        />

        <DataTableFacetedFilter
          column={table.getColumn("event")}
          title="Events"
          options={events.map((event) => ({
            label: event,
            value: event,
          }))}
        />

        {isFiltered && (
          <Button
            variant="ghost"
            onClick={() => table.resetColumnFilters()}
            className="h-8 px-2 lg:px-3"
          >
            Reset
            <X className="w-4 h-4 ml-2" />
          </Button>
        )}
      </div>
      <DataTableViewOptions table={table} />
    </div>
  );
}

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
];

export function DataTableToolbar<TData>({ table }: DataTableToolbarProps<TData>) {
  const isFiltered = table.getState().columnFilters.length > 0;

  return (
    <div className="flex items-center justify-between">
      <div className="flex flex-1 items-center space-x-2">
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

        {/* {table.getColumn("priority") && (
          <DataTableFacetedFilter
            column={table.getColumn("priority")}
            title="Priority"
            options={priorities}
          />
        )}  */}
        {isFiltered && (
          <Button
            variant="ghost"
            onClick={() => table.resetColumnFilters()}
            className="h-8 px-2 lg:px-3"
          >
            Reset
            <X className="ml-2 h-4 w-4" />
          </Button>
        )}
      </div>
      <DataTableViewOptions table={table} />
    </div>
  );
}

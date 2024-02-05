"use client";

import { Table } from "@tanstack/react-table";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

import { Badge } from "@/components/ui/badge";
import { X } from "lucide-react";
import { CreateNewPermission } from "./create-new-permission";

type DataTableToolbarProps<TData> = {
  table: Table<TData>;
  data: TData[];
};

export function DataTableToolbar<TData>({ table, data }: DataTableToolbarProps<TData>) {
  const isFiltered = table.getState().columnFilters.length > 0;

  return (
    <div className="flex items-center justify-between">
      <div className="flex items-center flex-1 space-x-2">
        <Input
          placeholder="Filter names..."
          value={(table.getColumn("name")?.getFilterValue() as string) ?? ""}
          onChange={(event) => table.getColumn("name")?.setFilterValue(event.target.value)}
          className="h-8 w-[150px] lg:w-[250px]"
        />
        <Input
          placeholder="Filter keys..."
          value={(table.getColumn("key")?.getFilterValue() as string) ?? ""}
          onChange={(event) => table.getColumn("key")?.setFilterValue(event.target.value)}
          className="h-8 w-[150px] lg:w-[250px]"
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
      <div className="flex items-center gap-2">
        <Badge variant="secondary" className="h-8">
          {Intl.NumberFormat().format(data.length)} / {Intl.NumberFormat().format(50)} used{" "}
        </Badge>
        <CreateNewPermission
          key="create-new-permission"
          trigger={<Button variant="primary">Create New Permission</Button>}
        />
      </div>
    </div>
  );
}

"use client";

import { ColumnDef } from "@tanstack/react-table";

import { Badge } from "@/components/ui/badge";

import { DataTableColumnHeader } from "@/components/data-table/column-header";

export const columns: ColumnDef<{ id: string; name: string }>[] = [
  {
    accessorKey: "name",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
    filterFn: (row, _id, value) => row.original.name.includes(value),
    cell: ({ row }) => {
      return (
        <div className="flex flex-col space-x-2">
          <Badge variant="secondary" font="mono">
            {row.original.name}
          </Badge>
          {/* <p>{row.original.description}</p> */}
        </div>
      );
    },
  },
];

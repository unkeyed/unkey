"use client";

import { ColumnDef } from "@tanstack/react-table";

import { Badge } from "@/components/ui/badge";

import { DataTableColumnHeader } from "@/components/data-table";
import type { Role } from "@/lib/db";

export const columns: ColumnDef<Role & { permissions: string[] }>[] = [
  {
    accessorKey: "name",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
    filterFn: (row, _id, value) => row.original.name.includes(value),
    cell: ({ row }) => {
      return (
        <div className="flex flex-col">
          <span className="font-semibold text-content">{row.original.name}</span>
          <span className="text-xs text-content-subtle">{row.original.description}</span>
        </div>
      );
    },
  },
  {
    accessorKey: "key",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Key" />,
    filterFn: (row, _id, value) => row.original.key.includes(value),

    cell: ({ row }) => {
      return <Badge variant="secondary">{row.original.key}</Badge>;
    },
    enableColumnFilter: true,
    enableSorting: true,
  },
  {
    accessorKey: "id",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Permissions" />,
    cell: ({ row }) => {
      return (
        <ul className="flex flex-col gap-1">
          {row.original.permissions.map((p) => (
            <Badge key={p}>{p}</Badge>
          ))}
        </ul>
      );
    },

    enableSorting: false,
  },
];

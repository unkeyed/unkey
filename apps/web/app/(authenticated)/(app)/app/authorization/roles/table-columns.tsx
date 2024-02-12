"use client";

import { ColumnDef } from "@tanstack/react-table";

import { DataTableColumnHeader } from "@/components/data-table/column-header";
import type { Role } from "@/lib/db";
import { ChevronRight } from "lucide-react";
import Link from "next/link";

export const columns: ColumnDef<Role & { permissions: string[] }>[] = [
  {
    accessorKey: "name",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
    filterFn: (row, _id, value) => row.original.name.toLowerCase().includes(value.toLowerCase()),
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
    accessorKey: "id",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Permissions" />,
    cell: ({ row }) => {
      return (
        <div className="flex flex-col">
          {row.original.permissions.sort().map((p) => (
            <pre key={p} className="font-mono text-xs text-content-subtle">
              {p}
            </pre>
          ))}
        </div>
      );
    },
    enableSorting: false,
  },
  {
    accessorKey: "id",
    header: () => null,
    cell: ({ row }) => (
      <Link href={`/app/authorization/roles/${row.original.id}`}>
        <ChevronRight />
      </Link>
    ),
  },
];

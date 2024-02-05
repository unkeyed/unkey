"use client";

import { ColumnDef } from "@tanstack/react-table";

import { Badge } from "@/components/ui/badge";

import { DataTableColumnHeader } from "@/components/data-table";
import { PermissionToggle } from "./permission-toggle";

export const columns: ColumnDef<{ id: string; name: string; roleId: string; checked: boolean }>[] =
  [
    {
      accessorKey: "id",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Enabled" />,
      cell: ({ row }) => {
        return (
          <PermissionToggle
            roleId={row.original.roleId}
            permissionId={row.original.id}
            checked={row.original.checked}
          />
        );
      },
      enableColumnFilter: false,
      enableSorting: false,
    },
    {
      accessorKey: "name",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
      filterFn: (row, _id, value) => row.original.name.includes(value),
      cell: ({ row }) => {
        return (
          <Badge variant="secondary" font="mono">
            {row.original.name}
          </Badge>
        );
      },
    },
  ];

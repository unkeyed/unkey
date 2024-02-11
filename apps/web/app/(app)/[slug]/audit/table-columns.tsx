"use client";

import { ColumnDef } from "@tanstack/react-table";

import { Badge } from "@/components/ui/badge";

import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { AuditLog } from "@/lib/db";
import { Key, Minus, User } from "lucide-react";
import Link from "next/link";
import { DataTableColumnHeader } from "./table-column-header";

export const columns: ColumnDef<AuditLog>[] = [
  {
    accessorKey: "time",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Time" />,
    cell: ({ row }) => {
      return (
        <Tooltip>
          <TooltipTrigger className="flex flex-col">
            <span className="text-content">{row.original.time.toLocaleTimeString()}</span>
            <span className="text-content-subtle text-xs">
              {row.original.time.toLocaleDateString()}
            </span>
          </TooltipTrigger>
          <TooltipContent>
            <span className="text-content">{row.original.time.toISOString()}</span>
          </TooltipContent>
        </Tooltip>
      );
    },
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: "actorId",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Actor" />,
    cell: ({ row }) => {
      return (
        <div className="flex items-center">
          {row.original.actorType === "user" ? (
            <User className="mr-2 h-4 w-4 text-muted-foreground" />
          ) : (
            <Key className="mr-2 h-4 w-4 text-muted-foreground" />
          )}
          <span>{row.original.actorId}</span>
        </div>
      );
    },
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: "event",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Event" />,
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
    cell: ({ row }) => {
      return (
        <div className="flex space-x-2">
          <Badge variant="secondary">{row.original.event}</Badge>
        </div>
      );
    },
  },
  {
    accessorKey: "apiId",
    header: ({ column }) => <DataTableColumnHeader column={column} title="API" />,
    cell: ({ row }) => {
      const apiId = row.getValue<string | null>("apiId");
      return apiId ? (
        <Link href={`/app/keys/${apiId}`}>
          <Badge variant="secondary" size="sm">
            {apiId}
          </Badge>
        </Link>
      ) : (
        <Minus className="h-4 w-4 text-content-subtle" />
      );
    },

    enableSorting: false,
    enableHiding: true,
  },
  {
    accessorKey: "keyId",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Key" />,
    cell: ({ row }) => {
      const keyId = row.getValue<string | null>("keyId");
      return keyId ? (
        <Link href={`/app/keys/${keyId}`}>
          <Badge variant="secondary" size="sm">
            {keyId}
          </Badge>
        </Link>
      ) : (
        <Minus className="h-4 w-4 text-content-subtle" />
      );
    },

    enableSorting: false,
    enableHiding: true,
  },
  {
    accessorKey: "description",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Description" />,
    cell: ({ row }) => (
      <div className="text-sm text-content-subtle">{row.getValue("description")}</div>
    ),
    enableSorting: false,
    enableHiding: true,
  },
];

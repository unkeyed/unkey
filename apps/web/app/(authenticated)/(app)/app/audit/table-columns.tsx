"use client";

import { ColumnDef } from "@tanstack/react-table";

import { Badge } from "@/components/ui/badge";

import { Avatar, AvatarImage } from "@/components/ui/avatar";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { AuditLog } from "@/lib/db";
import { useClerk, useOrganization } from "@clerk/nextjs";
import { AvatarFallback } from "@radix-ui/react-avatar";
import { Minus } from "lucide-react";
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
            <span className="text-xs text-content-subtle">
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
      // if (row.original.actorType === "user") {
      //   return (
      //     <Avatar className="w-8 h-8">
      //       <AvatarImage src={user.imageUrl} alt="Profile picture" />
      //       <AvatarFallback className="w-8 h-8 overflow-hidden text-gray-700 bg-gray-100 border border-gray-500 rounded-md ">
      //         {/* {(user?.firstName ?? "U").slice(0, 2).toUpperCase()} */}
      //       </AvatarFallback>
      //     </Avatar>
      //   );
      // }
      return (
        <div className="flex items-center">
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
        <Minus className="w-4 h-4 text-content-subtle" />
      );
    },

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
        <Minus className="w-4 h-4 text-content-subtle" />
      );
    },

    enableHiding: true,
  },
  {
    accessorKey: "ipAddress",
    header: ({ column }) => <DataTableColumnHeader column={column} title="IP address" />,
    cell: ({ row }) => {
      return row.original.ipAddress ? (
        <pre className="text-xs text-content-subtle">{row.original.ipAddress}</pre>
      ) : (
        <Minus className="w-4 h-4 text-content-subtle" />
      );
    },

    enableHiding: true,
  },
  {
    accessorKey: "description",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Description" />,
    cell: ({ row }) => (
      <div className="text-sm text-content-subtle">{row.getValue("description")}</div>
    ),
    enableHiding: true,
  },
];

const UserCell: React.FC<{ userId: string }> = ({ userId }) => {
  const { memberships } = useOrganization({
    memberships: { infinite: true },
  });
  return <div>Hello</div>;
};

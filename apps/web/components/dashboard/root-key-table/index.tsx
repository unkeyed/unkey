"use client";

import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { toast } from "@/components/ui/toaster";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { trpc } from "@/lib/trpc/client";
import { ColumnDef } from "@tanstack/react-table";
import { ArrowUpDown, Minus, MoreHorizontal, MoreVertical, Trash } from "lucide-react";
import ms from "ms";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Loading } from "../loading";
import { DataTable } from "./table";
type Column = {
  id: string;
  start: string;
  createdAt: Date;
  expires: Date | null;
  ownerId: string | null;
  name: string | null;
  ratelimitType: string | null;
  ratelimitLimit: number | null;
  ratelimitRefillRate: number | null;
  ratelimitRefillInterval: number | null;
  remaining: number | null;
};

type Props = {
  data: Column[];
};

export const RootKeyTable: React.FC<Props> = ({ data }) => {
  const router = useRouter();
  const deleteKey = trpc.key.deleteRootKey.useMutation({
    onSuccess: () => {
      toast.success("Root Key was deleted");
      router.refresh();
    },
    onError: (err, variables) => {
      toast(`Could not delete key ${JSON.stringify(variables)}`, {
        description: err.message,
      });
      router.refresh();
    },
  });

  const columns: ColumnDef<Column>[] = [
    {
      id: "select",

      header: ({ table }) => (
        <div className="flex items-center justify-center">
          <Checkbox
            checked={table.getIsAllPageRowsSelected()}
            onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
            aria-label="Select all"
          />
        </div>
      ),
      cell: ({ row }) => (
        <div className="flex items-center justify-center">
          <Checkbox
            checked={row.getIsSelected()}
            onCheckedChange={(value) => row.toggleSelected(!!value)}
            aria-label="Select row"
          />
        </div>
      ),
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: "start",
      header: "Key",
      cell: ({ row }) => (
        <Tooltip>
          <TooltipTrigger>
            <Badge variant="secondary">{row.getValue("start")}...</Badge>
          </TooltipTrigger>
          <TooltipContent>
            This is the first part of the key to visually match it. We don't store the full key for
            security reasons.
          </TooltipContent>
        </Tooltip>
      ),
    },
    {
      accessorKey: "createdAt",
      header: ({ column }) => (
        <Button
          variant="ghost"
          onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
        >
          Created At
          <ArrowUpDown className="w-4 h-4 ml-2" />
        </Button>
      ),
      cell: ({ row }) => row.original.createdAt.toUTCString(),
    },
    {
      accessorKey: "expires",
      header: "Expires",
      cell: ({ row }) =>
        row.original.expires ? (
          row.original.expires.getTime() < Date.now() ? (
            <span>Expired</span>
          ) : (
            <span>in {ms(row.original.expires.getTime() - Date.now(), { long: true })}</span>
          )
        ) : (
          <Minus className="w-4 h-4 text-gray-300" />
        ),
    },
    {
      accessorKey: "remaining",
      header: "Remaining",
      cell: ({ row }) =>
        row.original.remaining ? (
          <span>{row.original.remaining.toLocaleString()}</span>
        ) : (
          <Minus className="w-4 h-4 text-gray-300" />
        ),
    },
    {
      accessorKey: "ownerId",
      header: "Owner",
      cell: ({ row }) =>
        row.original.ownerId ? (
          <Badge variant="secondary">{row.original.ownerId}</Badge>
        ) : (
          <Minus className="w-4 h-4 text-gray-300" />
        ),
    },
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) =>
        row.original.name ? (
          <Badge variant="secondary">{row.original.name}</Badge>
        ) : (
          <Minus className="w-4 h-4 text-gray-300" />
        ),
    },
    {
      accessorKey: "ratelimit",
      header: "Ratelimit",
      cell: ({ row }) =>
        row.original.ratelimitType &&
        row.original.ratelimitLimit &&
        row.original.ratelimitRefillInterval &&
        row.original.ratelimitRefillRate ? (
          <div>
            <span>{row.original.ratelimitRefillRate}</span> /{" "}
            <span>{ms(row.original.ratelimitRefillInterval)}</span>
          </div>
        ) : (
          <Minus className="w-4 h-4 text-gray-300" />
        ),
    },
    {
      id: "actions",
      cell: ({ row }) => (
        <div>
          <Dialog>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" className="w-8 h-8 p-0">
                  <span className="sr-only">Open menu</span>
                  <MoreHorizontal className="w-4 h-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem
                  onSelect={(e) => {
                    e.preventDefault();
                  }}
                >
                  <MoreVertical className="w-4 h-4 mr-2" />
                  <Link href={`/app/settings/root-keys/${row.original.id}`}>Details</Link>
                </DropdownMenuItem>
                <DialogTrigger asChild>
                  <DropdownMenuItem
                    onSelect={(e) => {
                      e.preventDefault();
                    }}
                  >
                    <Trash className="w-4 h-4 mr-2" />
                    Revoke Key
                  </DropdownMenuItem>
                </DialogTrigger>
                <DialogContent className="sm:max-w-[425px]">
                  <DialogHeader>
                    <DialogTitle>Revoke Root Key</DialogTitle>
                    <DialogDescription>
                      Delete the key <Badge variant="secondary">{row.original.start}...</Badge>{" "}
                      permanenty
                    </DialogDescription>
                    <Alert variant="alert">
                      This action can not be undone. Your root key will no longer be able to create
                      resources
                    </Alert>
                  </DialogHeader>

                  <DialogFooter>
                    <Button
                      variant="alert"
                      disabled={deleteKey.isLoading}
                      onClick={() => deleteKey.mutate({ keyIds: [row.original.id] })}
                    >
                      {" "}
                      {deleteKey.isLoading ? <Loading /> : "Delete permanently"}
                    </Button>
                  </DialogFooter>
                </DialogContent>
              </DropdownMenuContent>
            </DropdownMenu>
          </Dialog>
        </div>
      ),
    },
  ];

  return <DataTable columns={columns} data={data} />;
};

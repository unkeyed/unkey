"use client";

import { DataTable } from "./table";
import { ColumnDef } from "@tanstack/react-table";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Checkbox } from "@/components/ui/checkbox";
import { ArrowUpDown, Minus, MoreHorizontal, Trash } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Alert } from "@/components/ui/alert";
import { trpc } from "@/lib/trpc/client";
import { Loading } from "../loading";
import { useRouter } from "next/navigation";
import { useToast } from "@/components/ui/use-toast";
import ms from "ms";
import Link from "next/link";
type Column = {
  id: string;
  apiId: string | null;
  start: string;
  createdAt: Date;
  expires: Date | null;
  ownerId: string | null;
  ratelimitType: string | null;
  ratelimitLimit: number | null;
  ratelimitRefillRate: number | null;
  ratelimitRefillInterval: number | null;
};

type Props = {
  data: Column[];
};

export const ApiKeyTable: React.FC<Props> = ({ data }) => {
  const router = useRouter();
  const { toast } = useToast();
  const deleteKey = trpc.key.delete.useMutation({
    onSuccess: () => {
      toast({
        title: "Key was deleted",
      });
      router.refresh();
    },
    onError: (err, variables) => {
      toast({
        title: `Could not delete key ${JSON.stringify(variables)}`,
        description: err.message,
        variant: "default",
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
      cell: ({ row }) => <Badge variant="secondary">{row.getValue("start")}...</Badge>,
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
          <span>in {ms(row.original.expires.getTime() - Date.now(), { long: true })}</span>
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
                  <Link
                    href={`/app/${row.original.apiId}/keys/${row.original.id}`}
                    className="w-full"
                  >
                    Details
                  </Link>
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
                    <DialogTitle>Revoke Api Key</DialogTitle>
                    <DialogDescription>
                      Delete the key <Badge variant="outline">{row.original.start}...</Badge>{" "}
                      permanenty
                    </DialogDescription>
                    <Alert variant="destructive">
                      This action can not be undone. Your users will no longer be able to
                      authenticate using this key.
                    </Alert>
                  </DialogHeader>

                  <DialogFooter>
                    <Button
                      variant="destructive"
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

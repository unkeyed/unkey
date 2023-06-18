"use client";

import { DataTable } from "./table";
import { ColumnDef } from "@tanstack/react-table";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuPortal,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
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
import { Loading } from "@/components/loading";
import { useRouter } from "next/navigation";
import { useToast } from "@/components/ui/use-toast";
type Column = {
  id: string;
  start: string;
  createdAt: Date;
  expires: Date | null;
  ownerId: string | null;
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

  const columns: ColumnDef<{
    id: string;
    start: string;
    createdAt: Date;
    expires: Date | null;
    ownerId: string | null;
  }>[] = [
    {
      id: "select",
      header: ({ table }) => (
        <Checkbox
          checked={table.getIsAllPageRowsSelected()}
          onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
          aria-label="Select all"
        />
      ),
      cell: ({ row }) => (
        <Checkbox
          checked={row.getIsSelected()}
          onCheckedChange={(value) => row.toggleSelected(!!value)}
          aria-label="Select row"
        />
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
        row.original.expires?.toUTCString() ?? <Minus className="w-4 h-4 text-gray-300" />,
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

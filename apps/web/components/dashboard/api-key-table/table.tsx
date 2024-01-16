"use client";

import {
  ColumnDef,
  ColumnFiltersState,
  SortingState,
  VisibilityState,
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from "@tanstack/react-table";

import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

import { Button } from "@/components/ui/button";
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
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { useRouter } from "next/navigation";
import { useState } from "react";
interface DataTableProps<TData, TValue> {
  columns: ColumnDef<TData, TValue>[];
  data: TData[];
}

export function DataTable<TData, TValue>({ data, columns }: DataTableProps<TData, TValue>) {
  const [sorting, setSorting] = useState<SortingState>([]);
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
  const [rowSelection, setRowSelection] = useState({});
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});
  const router = useRouter();
  const deleteKey = trpc.key.delete.useMutation({
    onSuccess: (_data, variables) => {
      setRowSelection({});
      toast(
        variables.keyIds.length > 1
          ? `All ${variables.keyIds.length} keys were deleted`
          : "Key deleted",
      );
      router.refresh();
    },
    onError: (err, variables) => {
      router.refresh();
      toast.error(`Could not delete key ${JSON.stringify(variables)}`, {
        description: err.message,
      });
    },
  });
  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    onSortingChange: setSorting,
    onColumnVisibilityChange: setColumnVisibility,

    getSortedRowModel: getSortedRowModel(),
    state: {
      sorting,
      columnFilters,
      rowSelection,
      columnVisibility,
    },
    onColumnFiltersChange: setColumnFilters,
    getFilteredRowModel: getFilteredRowModel(),
    onRowSelectionChange: setRowSelection,
  });

  return (
    <div>
      <div>
        <div className="flex flex-wrap items-center justify-between space-x-0 space-y-4 md:space-x-4 sm:space-y-0 sm:flex-nowrap">
          {Object.values(rowSelection).length > 0 ? (
            <Dialog>
              <DialogTrigger asChild>
                <Button className="min-w-full md:min-w-min " variant="secondary">
                  Revoke selected keys
                </Button>
              </DialogTrigger>
              <DialogContent className="sm:max-w-[425px]">
                <DialogHeader>
                  <DialogTitle>Revoke {Object.keys(rowSelection).length} keys</DialogTitle>
                  <DialogDescription className="text-alert">
                    This action can not be undone. Your root key(s) will no longer be able to create
                    resources
                  </DialogDescription>
                </DialogHeader>

                <DialogFooter>
                  <Button
                    onClick={() => {
                      const keyIds = table
                        .getSelectedRowModel()
                        // @ts-ignore
                        .rows.map((row) => row.original.id as string);
                      deleteKey.mutate({ keyIds });
                    }}
                  >
                    Delete permanently
                  </Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          ) : null}
          <Input
            placeholder="Filter keys"
            value={(table.getColumn("start")?.getFilterValue() as string) ?? ""}
            onChange={(event) => table.getColumn("start")?.setFilterValue(event.target.value)}
            className="max-w-sm md:max-w-2xl"
          />
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                className="min-w-full mt-4 ml-0 md:min-w-min md:mt-0 md:ml-auto"
                variant="secondary"
              >
                Columns
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              {table
                .getAllColumns()
                .filter((column) => column.getCanHide())
                .map((column) => {
                  return (
                    <DropdownMenuCheckboxItem
                      key={column.id}
                      className="capitalize"
                      checked={column.getIsVisible()}
                      onCheckedChange={(value) => column.toggleVisibility(!!value)}
                    >
                      {column.id}
                    </DropdownMenuCheckboxItem>
                  );
                })}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
      <Table className="mt-4">
        <TableHeader>
          {table.getHeaderGroups().map((headerGroup) => (
            <TableRow key={headerGroup.id}>
              {headerGroup.headers.map((header) => {
                return (
                  <TableHead key={header.id}>
                    {header.isPlaceholder
                      ? null
                      : flexRender(header.column.columnDef.header, header.getContext())}
                  </TableHead>
                );
              })}
            </TableRow>
          ))}
        </TableHeader>
        <TableBody>
          {table.getRowModel().rows?.length ? (
            table.getRowModel().rows.map((row) => (
              <TableRow key={row.id} data-state={row.getIsSelected() && "selected"}>
                {row.getVisibleCells().map((cell) => (
                  <TableCell key={cell.id}>
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </TableCell>
                ))}
              </TableRow>
            ))
          ) : (
            <TableRow>
              <TableCell colSpan={columns.length} className="h-24 text-center">
                No results.
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
      <div className="flex items-center justify-end py-4 space-x-2">
        <Button
          variant={table.getCanPreviousPage() ? "secondary" : "disabled"}
          size="sm"
          onClick={() => table.previousPage()}
          disabled={!table.getCanPreviousPage()}
        >
          Previous
        </Button>
        <Button
          variant={table.getCanNextPage() ? "secondary" : "disabled"}
          size="sm"
          onClick={() => table.nextPage()}
          disabled={!table.getCanNextPage()}
        >
          Next
        </Button>
      </div>
    </div>
  );
}

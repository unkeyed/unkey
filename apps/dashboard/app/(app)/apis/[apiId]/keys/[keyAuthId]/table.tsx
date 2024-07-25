"use client";

import {
  type ColumnDef,
  type ColumnFiltersState,
  type Row,
  type SortingState,
  type VisibilityState,
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from "@tanstack/react-table";
import { Database, DatabaseZap, Minus } from "lucide-react";
import * as React from "react";
import { IntervalSelect } from "../../../apis/[apiId]/select";
import { Dialog, DialogContent, DialogOverlay, DialogTrigger } from "../../components/sidebar";

import { Button } from "@/components/ui/button";
import { Code } from "@/components/ui/code";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

export function KeysTable({ data, defaultInterval }: { data: Event[]; defaultInterval: string }) {
  const [sorting, setSorting] = React.useState<SortingState>([]);
  const [columnFilters, setColumnFilters] = React.useState<ColumnFiltersState>([]);
  const [columnVisibility, setColumnVisibility] = React.useState<VisibilityState>({});
  const [rowSelection, setRowSelection] = React.useState({});
  const [timeRange, setTimeRange] = React.useState(defaultInterval);

  const columns: ColumnDef<Event>[] = [
    {
      accessorKey: "id",
      header: "Time",
      cell: ({ row }) => {
        const time = row.getValue("time") as number;
        const date = new Date(time);
        let timeOptions: Intl.DateTimeFormatOptions;
        switch (timeRange) {
          case "24h":
            timeOptions = {
              hour: "2-digit",
              minute: "2-digit",
              second: "2-digit",
              hour12: false,
            };
            break;
          default:
            timeOptions = {
              month: "long",
              day: "numeric",
              hour: "2-digit",
              minute: "2-digit",
              second: "2-digit",
            };
            break;
        }
        const formattedTime = new Intl.DateTimeFormat(undefined, timeOptions).format(date);
        return (
          <>
            <div className="">{formattedTime}</div>
          </>
        );
      },
    },
    {
      accessorKey: "verifications",
      header: "Verifications",
      cell: ({ row }) => {
        const latency = row.getValue("serviceLatency") as number;
        return <div>{latency}ms</div>;
      },
    },
    {
      accessorKey: "lastUsed",
      header: "Last Used",
    },
    {
      accessorKey: "ownerId",
      header: "Owner ID",
      cell: ({ row }) => {
        const cache = row.getValue("cache") as number;
        return (
          <div className="flex items-center">
            {cache ? (
              <DatabaseZap className="w-5 h-5 ml-2" />
            ) : (
              <Database className="w-5 h-5 ml-2 text-gray-600" />
            )}
          </div>
        );
      },
    },
    {
      accessorKey: "chart",
      header: "Model",
    },
  ];

  const table = useReactTable({
    data,
    columns,
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    manualPagination: true,
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    onColumnVisibilityChange: setColumnVisibility,
    onRowSelectionChange: setRowSelection,
    state: {
      sorting,
      columnFilters,
      columnVisibility,
      rowSelection,
    },
  });

  const [rowID, setRowID] = React.useState(table.getRowModel().rows[0]?.id);

  return (
    <div className="mt-4 ml-1 mb-">
      <div className="flex justify-between">
        <div className="flex mb-2 space-x-3 md:w-full">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" className="ml-auto">
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
          <IntervalSelect
            defaultSelected="7d"
            className="w-[200px]"
            onChange={(i: string) => setTimeRange(`${i}`)}
          />
        </div>
      </div>
      <div className="w-full">
        <Dialog>
          <Table>
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
                  <DialogTrigger asChild>
                    <TableRow
                      key={row.id}
                      className="cursor-pointer"
                      onClick={() => setRowID(row.id)}
                    >
                      {row.getVisibleCells().map((cell) => (
                        <>
                          <TableCell key={cell.id} className="px-4 py-2">
                            {flexRender(cell.column.columnDef.cell, cell.getContext())}
                          </TableCell>
                        </>
                      ))}
                    </TableRow>
                  </DialogTrigger>
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
          <DialogOverlay className="bg-red-500">
            {table.getRowModel().rows?.length ? (
              <DialogContent className="sm:max-w-[425px] transform-none left-[unset] right-0 top-0 min-h-full overflow-y-auto flex flex-col">
                <p className="font-medium text-gray-300">Request ID:</p>
                <Code>
                  <pre>{table.getRow(rowID).original.requestId}</pre>
                </Code>
                <p className="font-medium text-gray-300">Query:</p>
                <Code>
                  <pre>{table.getRow(rowID).original.query}</pre>
                </Code>
                <p className="font-medium text-gray-300">Response:</p>
                <Code className="max-w-sm whitespace-pre-wrap">
                  {table.getRow(rowID).original.response}
                </Code>
              </DialogContent>
            ) : null}
          </DialogOverlay>
        </Dialog>
      </div>
    </div>
  );
}

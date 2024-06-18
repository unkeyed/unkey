"use client";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { Workspace } from "@/lib/db";
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
import { Database, DatabaseZap } from "lucide-react";
import * as React from "react";
import { IntervalSelect } from "../../apis/[apiId]/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogOverlay,
  DialogTitle,
  DialogTrigger,
} from "../components/sidebar";

import { download, generateCsv, mkConfig } from "export-to-csv";

type Event = {
  requestId: string;
  time: number;
  serviceLatency: number;
  embeddingsLatency: number;
  vectorizeLatency: number;
  inferenceLatency?: number;
  cacheLatency: number;
  gatewayId: string;
  workspaceId: string;
  stream: number;
  tokens: number;
  cache: number;
  model: string;
  query: string;
  vector: any[]; // Replace `any` with the specific type if known
  response: string;
};

export const columns: ColumnDef<Event>[] = [
  {
    accessorKey: "time",
    header: "Time",
    cell: ({ row }) => {
      const time = row.getValue("time") as number;
      const date = new Date(time);
      const options: Intl.DateTimeFormatOptions = {
        month: "long",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
      };
      const formattedTime = date.toLocaleDateString("en-US", options);
      return <div>{formattedTime}</div>;
    },
  },
  {
    accessorKey: "serviceLatency",
    header: "Latency",
    cell: ({ row }) => {
      const latency = row.getValue("serviceLatency") as number;
      return <div>{latency}ms</div>;
    },
  },
  {
    accessorKey: "tokens",
    header: "Tokens",
  },

  {
    accessorKey: "cache",
    header: "Cache",
    cell: ({ row }) => {
      const cache = row.getValue("cache") as number;
      return (
        <div className="flex items-center">
          {cache ? (
            <DatabaseZap className="ml-2 h-5 w-5" />
          ) : (
            <Database className="ml-2 h-5 w-5 text-gray-600" />
          )}
        </div>
      );
    },
  },
  {
    accessorKey: "model",
    header: "Model",
  },
  {
    accessorKey: "query",
    header: "Query",
  },
];

const omitProperty = <T extends Record<string, any>, K extends keyof T>(
  obj: T,
  key: K,
): Omit<T, K> => {
  const { [key]: omitted, ...rest } = obj;
  return rest;
};

function exportToCsv(rows: Row<Event>[]) {
  const csvConfig = mkConfig({
    fieldSeparator: ",",
    filename: "semantic-cache-logs", // export file name (without .csv)
    decimalSeparator: ".",
    useKeysAsHeaders: true,
  });
  const rowData = rows.map((row) => omitProperty(row.original, "vector"));

  const csv = generateCsv(csvConfig)(rowData);
  download(csvConfig)(csv);
}

export function LogsTable({ data }: { data: Event[]; workspace: Workspace }) {
  const [sorting, setSorting] = React.useState<SortingState>([]);
  const [columnFilters, setColumnFilters] = React.useState<ColumnFiltersState>([]);
  const [columnVisibility, setColumnVisibility] = React.useState<VisibilityState>({});
  const [rowSelection, setRowSelection] = React.useState({});
  const [rowID, setRowID] = React.useState<string>("0");

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

  const row = table.getRow(rowID);

  return (
    <div className="mt-4 ml-1 mb-">
      <div className="flex justify-between">
        <h1 className="font-medium">Logs</h1>
        <div className="flex space-x-3 mb-2">
          {/* <Input
            placeholder="Filter responses..."
            value={(table.getColumn("email")?.getFilterValue() as string) ?? ""}
            onChange={(event) => table.getColumn("email")?.setFilterValue(event.target.value)}
          /> */}
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
          <IntervalSelect defaultSelected="7d" className="w-[200px]" />
          <Button variant="outline" onClick={() => exportToCsv(table.getFilteredRowModel().rows)}>
            Export
          </Button>
        </div>
      </div>
      <div className="w-full">
        <div className="rounded-md border">
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
              <DialogContent className="sm:max-w-[425px] transform-none left-[unset] right-0 top-0 h-full">
                <p className="text-white">{row.original.response}</p>
              </DialogContent>
            </DialogOverlay>
          </Dialog>
        </div>
      </div>
    </div>
  );
}

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

import { download, generateCsv, mkConfig } from "export-to-csv";
import ms from "ms";

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
  vector: any[];
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
      return <div>{ms(row.getValue("serviceLatency"))}</div>;
    },
  },
  {
    accessorKey: "tokens",
    header: "Tokens",
    cell: ({ row }) => {
      return (
        <div>
          {Intl.NumberFormat(undefined, { notation: "compact" }).format(row.getValue("tokens"))}
        </div>
      );
    },
  },

  {
    accessorKey: "cache",
    header: "Cache",
    cell: ({ row }) => {
      const cache = row.getValue("cache") as number;
      return (
        <div className="flex items-center">
          {cache ? (
            <DatabaseZap className="ml-2.5 h-4 w-4 text-content" />
          ) : (
            <Minus className="ml-2.5 h-4 w-4 text-content-subtle" />
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

export function LogsTable({ data, defaultInterval }: { data: Event[]; defaultInterval: string }) {
  const [sorting, setSorting] = React.useState<SortingState>([]);
  const [columnFilters, setColumnFilters] = React.useState<ColumnFiltersState>([]);
  const [columnVisibility, setColumnVisibility] = React.useState<VisibilityState>({});
  const [rowSelection, setRowSelection] = React.useState({});
  const [timeRange, setTimeRange] = React.useState(defaultInterval);

  const columns: ColumnDef<Event>[] = [
    {
      accessorKey: "time",
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
              <DatabaseZap className="w-5 h-5 ml-2" />
            ) : (
              <Database className="w-5 h-5 ml-2 text-gray-600" />
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
      cell: ({ row }) => {
        const query = row.getValue("query") as string;
        return <div>{query}</div>;
      },
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
            onChange={(i) => setTimeRange(`${i}`)}
          />
          <Button variant="outline" onClick={() => exportToCsv(table.getFilteredRowModel().rows)}>
            Export
          </Button>
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
              <DialogContent className="sm:max-w-[425px] transform-none left-[unset] right-0 top-0 min-h-full overflow-y-auto">
                <p className="font-medium text-gray-300">Request ID:</p>
                <Code>
                  <pre>{table.getRow(rowID).original.requestId}</pre>
                </Code>
                <p className="font-medium text-gray-300">Query:</p>
                <Code>
                  <pre>{table.getRow(rowID).original.query}</pre>
                </Code>
                <p className="font-medium text-gray-300">Response:</p>
                <Code>
                  <pre>
                    {" "}
                    {table
                      .getRow(rowID)
                      .original.response.split("\\n")
                      .map((line) => (
                        <span key={table.getRow(rowID).original.requestId}>
                          {line}
                          <br />
                        </span>
                      ))}
                  </pre>
                </Code>
              </DialogContent>
            ) : null}
          </DialogOverlay>
        </Dialog>
      </div>
    </div>
  );
}

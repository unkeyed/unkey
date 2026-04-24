import {
  type ColumnFiltersState,
  type Header,
  type OnChangeFn,
  type SortingState,
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from "@tanstack/react-table";
import { ArrowDown, ArrowUp, ArrowUpDown, KeyRound, Plus } from "lucide-react";
import { useMemo, useState } from "react";
import { CreateKeySheet } from "~/components/keys-table/create-key-sheet";
import { createKeysColumns, globalSearchFn } from "~/components/keys-table/keys-columns";
import { KeysPagination } from "~/components/keys-table/keys-pagination";
import { KeysToolbar, type StatusFilter } from "~/components/keys-table/keys-toolbar";
import { Button } from "~/components/ui/button";
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "~/components/ui/empty";
import { Sheet, SheetTrigger } from "~/components/ui/sheet";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "~/components/ui/table";
import type { Key } from "~/routes/dave-initial-design/-seed";

const PAGE_SIZE = 25;

type Props = {
  appName: string;

  keys: Key[];
  totalCount?: number;
  manualPagination?: boolean;

  searchValue: string;
  onSearchChange: (value: string) => void;
  statusValue: StatusFilter;
  onStatusChange: (value: StatusFilter) => void;
  sorting: SortingState;
  onSortingChange: OnChangeFn<SortingState>;
  pageIndex: number;
  onPageChange: (index: number) => void;

  onDelete?: (id: string) => void;
  onEditExpiration?: (id: string) => void;
  onRotate?: (id: string) => void;
};

export function KeysTable({
  appName,
  keys,
  totalCount,
  manualPagination,
  searchValue,
  onSearchChange,
  statusValue,
  onStatusChange,
  sorting,
  onSortingChange,
  pageIndex,
  onPageChange,
  onDelete,
  onEditExpiration,
  onRotate,
}: Props) {
  const columns = useMemo(
    () => createKeysColumns({ onDelete, onEditExpiration, onRotate }),
    [onDelete, onEditExpiration, onRotate],
  );

  const columnFilters = useMemo<ColumnFiltersState>(
    () => (statusValue === "all" ? [] : [{ id: "status", value: statusValue }]),
    [statusValue],
  );

  const table = useReactTable<Key>({
    data: keys,
    columns,
    state: {
      sorting,
      columnFilters,
      globalFilter: searchValue,
      pagination: { pageIndex, pageSize: PAGE_SIZE },
    },
    onSortingChange,
    onGlobalFilterChange: (updater) => {
      const next = typeof updater === "function" ? updater(searchValue) : updater;
      onSearchChange(typeof next === "string" ? next : "");
    },
    onColumnFiltersChange: (updater) => {
      const next = typeof updater === "function" ? updater(columnFilters) : updater;
      const found = next.find((f) => f.id === "status")?.value;
      onStatusChange((found as StatusFilter | undefined) ?? "all");
    },
    onPaginationChange: (updater) => {
      const current = { pageIndex, pageSize: PAGE_SIZE };
      const next = typeof updater === "function" ? updater(current) : updater;
      onPageChange(next.pageIndex);
    },
    globalFilterFn: globalSearchFn,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    manualPagination,
    rowCount: manualPagination ? totalCount : undefined,
  });

  const [createOpen, setCreateOpen] = useState(false);

  const visibleRows = table.getRowModel().rows;
  const showNoKeys = keys.length === 0;
  const showNoMatches = !showNoKeys && visibleRows.length === 0;

  const handleClearFilters = () => {
    onSearchChange("");
    onStatusChange("all");
    onPageChange(0);
  };

  return (
    <section className="flex flex-col gap-6">
      <header className="flex items-start justify-between gap-4">
        <div className="flex flex-col gap-1">
          <h1 className="font-semibold text-gray-12 text-xl">{appName} API</h1>
          <p className="text-gray-11 text-sm">
            Manage the API keys you use to authenticate with {appName}.
          </p>
        </div>
        <Sheet open={createOpen} onOpenChange={setCreateOpen}>
          <SheetTrigger asChild>
            <Button>
              <Plus />
              Create key
            </Button>
          </SheetTrigger>
          <CreateKeySheet appName={appName} />
        </Sheet>
      </header>

      <div className="overflow-hidden rounded-lg border border-primary/10 bg-background">
        {!showNoKeys && (
          <div className="flex items-center justify-between gap-2 border-primary/10 border-b p-3">
            <KeysToolbar
              searchValue={searchValue}
              onSearchChange={(v) => {
                onSearchChange(v);
                onPageChange(0);
              }}
              statusValue={statusValue}
              onStatusChange={(v) => {
                onStatusChange(v);
                onPageChange(0);
              }}
            />
            <KeysPagination table={table} />
          </div>
        )}
        {showNoKeys ? (
          <KeysEmptyState
            title="No API keys yet"
            description="Create your first key to start making authenticated requests."
            action={
              <Button onClick={() => setCreateOpen(true)}>
                <Plus />
                Create key
              </Button>
            }
          />
        ) : showNoMatches ? (
          <KeysEmptyState
            title="No keys match your filters"
            description="Try a different search term or status filter."
            action={
              <Button variant="ghost" onClick={handleClearFilters}>
                Clear filters
              </Button>
            }
          />
        ) : (
          <Table>
            <TableHeader>
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id} className="bg-gray-2 hover:bg-gray-2">
                  {headerGroup.headers.map((header) => (
                    <TableHead
                      key={header.id}
                      className={header.column.columnDef.meta?.className}
                    >
                      {header.isPlaceholder ? null : <SortableHeader header={header} />}
                    </TableHead>
                  ))}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody>
              {visibleRows.map((row) => (
                <TableRow key={row.id} className="h-14">
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id}>
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </TableCell>
                  ))}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </div>
    </section>
  );
}

function KeysEmptyState({
  title,
  description,
  action,
}: {
  title: string;
  description: string;
  action: React.ReactNode;
}) {
  return (
    <Empty>
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <KeyRound />
        </EmptyMedia>
        <EmptyTitle>{title}</EmptyTitle>
        <EmptyDescription>{description}</EmptyDescription>
      </EmptyHeader>
      <EmptyContent>{action}</EmptyContent>
    </Empty>
  );
}

function SortableHeader({ header }: { header: Header<Key, unknown> }) {
  const content = flexRender(header.column.columnDef.header, header.getContext());
  if (!header.column.getCanSort()) return content;

  const sortDir = header.column.getIsSorted();
  return (
    <button
      type="button"
      onClick={header.column.getToggleSortingHandler()}
      className="group inline-flex items-center gap-1 hover:text-gray-12"
    >
      {content}
      {sortDir === "asc" ? (
        <ArrowUp className="size-3 text-gray-12" />
      ) : sortDir === "desc" ? (
        <ArrowDown className="size-3 text-gray-12" />
      ) : (
        <ArrowUpDown className="size-3 text-gray-9 opacity-0 transition-opacity group-hover:opacity-100" />
      )}
    </button>
  );
}

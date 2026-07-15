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
import { CreateKeyDialog } from "~/components/keys-table/create-key-dialog";
import { DeleteKeyDialog } from "~/components/keys-table/delete-key-dialog";
import { EditKeyDialog, type EditKeyValues } from "~/components/keys-table/edit-key-dialog";
import { createKeysColumns, globalSearchFn } from "~/components/keys-table/keys-columns";
import { KeysPagination } from "~/components/keys-table/keys-pagination";
import { KeysToolbar, type StatusFilter } from "~/components/keys-table/keys-toolbar";
import { RotateKeyDialog, type RotateResult } from "~/components/keys-table/rotate-key-dialog";
import { Button } from "~/components/ui/button";
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "~/components/ui/empty";
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

function parseStatusFilter(value: unknown): StatusFilter {
  return value === "enabled" || value === "disabled" || value === "expired" ? value : "all";
}

type Props = {
  appName: string;

  keys: Key[];

  searchValue: string;
  onSearchChange: (value: string) => void;
  statusValue: StatusFilter;
  onStatusChange: (value: StatusFilter) => void;
  sorting: SortingState;
  onSortingChange: OnChangeFn<SortingState>;
  pageIndex: number;
  onPageChange: (index: number) => void;

  onDelete?: (id: string) => void;
  onEdit?: (id: string, values: EditKeyValues) => void;
  onRotate?: (id: string, result: RotateResult) => void;
  onCreate: (key: Key) => void;
  freshKeyId?: string | null;
};

export function KeysTable({
  appName,
  keys,
  searchValue,
  onSearchChange,
  statusValue,
  onStatusChange,
  sorting,
  onSortingChange,
  pageIndex,
  onPageChange,
  onDelete,
  onEdit,
  onRotate,
  onCreate,
  freshKeyId,
}: Props) {
  const [pendingDeleteKey, setPendingDeleteKey] = useState<Key | null>(null);
  const [pendingEditKey, setPendingEditKey] = useState<Key | null>(null);
  const [pendingRotateKey, setPendingRotateKey] = useState<Key | null>(null);
  const [exitingId, setExitingId] = useState<string | null>(null);

  const columns = useMemo(() => {
    // Defer to the next frame so the DropdownMenu's body pointer-events lock is
    // released before AlertDialog mounts; otherwise Radix captures
    // `pointer-events: none` as the body's original style and restores it on
    // close, leaving the page uninteractable.
    const openWithKey = (id: string, set: (k: Key) => void) => {
      const key = keys.find((k) => k.id === id);
      if (key) {
        requestAnimationFrame(() => set(key));
      }
    };
    return createKeysColumns({
      onDelete: (id) => openWithKey(id, setPendingDeleteKey),
      onEdit: (id) => openWithKey(id, setPendingEditKey),
      onRotate: (id) => openWithKey(id, setPendingRotateKey),
    });
  }, [keys]);

  const handleConfirmDelete = () => {
    if (!pendingDeleteKey) {
      return;
    }
    const id = pendingDeleteKey.id;
    setPendingDeleteKey(null);

    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
      onDelete?.(id);
      return;
    }

    setExitingId(id);
    window.setTimeout(() => {
      onDelete?.(id);
      setExitingId(null);
    }, 200);
  };

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
      onStatusChange(parseStatusFilter(found));
    },
    onPaginationChange: (updater) => {
      const current = { pageIndex, pageSize: PAGE_SIZE };
      const next = typeof updater === "function" ? updater(current) : updater;
      onPageChange(next.pageIndex);
    },
    globalFilterFn: globalSearchFn,
    getRowId: (row) => row.id,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
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
    <section className="flex flex-col gap-3">
      <header className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
        <div className="flex flex-col gap-1">
          <h1 className="font-semibold text-gray-12 text-xl">{appName} API</h1>
          <p className="text-gray-11 text-sm">
            Manage the API keys you use to authenticate with {appName}.
          </p>
        </div>
        <Button onClick={() => setCreateOpen(true)} className="self-start sm:self-auto">
          <Plus />
          Create key
        </Button>
      </header>

      <CreateKeyDialog open={createOpen} onOpenChange={setCreateOpen} onCreate={onCreate} />

      {!showNoKeys && (
        <div className="mt-3 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
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

      <div className="overflow-hidden rounded-lg border border-primary/10 bg-background">
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
          <Table className="min-w-[800px] table-fixed">
            <TableHeader>
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id} className="bg-gray-2 hover:bg-gray-2">
                  {headerGroup.headers.map((header) => (
                    <TableHead key={header.id} className={header.column.columnDef.meta?.className}>
                      {header.isPlaceholder ? null : <SortableHeader header={header} />}
                    </TableHead>
                  ))}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody>
              {visibleRows.map((row) => (
                <TableRow
                  key={row.id}
                  data-exiting={row.id === exitingId ? "true" : undefined}
                  data-fresh={row.id === freshKeyId ? "true" : undefined}
                  className="h-14 data-[exiting=true]:pointer-events-none motion-safe:transition-[opacity,filter,background-color] motion-safe:duration-200 motion-safe:ease-out data-[exiting=true]:motion-safe:opacity-0 data-[fresh=true]:motion-safe:opacity-0 data-[exiting=true]:motion-safe:blur-[2px] data-[fresh=true]:motion-safe:blur-[2px]"
                >
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id} className={cell.column.columnDef.meta?.className}>
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </TableCell>
                  ))}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </div>

      <DeleteKeyDialog
        open={pendingDeleteKey !== null}
        onOpenChange={(open) => {
          if (!open) {
            setPendingDeleteKey(null);
          }
        }}
        onConfirm={handleConfirmDelete}
      />

      <EditKeyDialog
        open={pendingEditKey !== null}
        onOpenChange={(open) => {
          if (!open) {
            setPendingEditKey(null);
          }
        }}
        keyToEdit={pendingEditKey}
        onSave={(id, values) => {
          onEdit?.(id, values);
          setPendingEditKey(null);
        }}
      />

      <RotateKeyDialog
        open={pendingRotateKey !== null}
        onOpenChange={(open) => {
          if (!open) {
            setPendingRotateKey(null);
          }
        }}
        keyToRotate={pendingRotateKey}
        onRotate={(id, result) => {
          onRotate?.(id, result);
          setPendingRotateKey(null);
        }}
      />
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
  if (!header.column.getCanSort()) {
    return content;
  }

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

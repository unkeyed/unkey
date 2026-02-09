import {
  type OnChangeFn,
  type RowSelectionState,
  type SortingState,
  getCoreRowModel,
  getSortedRowModel,
  useReactTable,
} from "@tanstack/react-table";
import { useMemo, useState } from "react";
import type { DataTableColumnDef } from "../types";

interface UseDataTableProps<TData> {
  data: TData[];
  columns: DataTableColumnDef<TData>[];
  getRowId: (row: TData) => string;
  enableSorting?: boolean;
  enableRowSelection?: boolean;
  sorting?: SortingState;
  onSortingChange?: OnChangeFn<SortingState>;
  rowSelection?: RowSelectionState;
  onRowSelectionChange?: OnChangeFn<RowSelectionState>;
}

/**
 * TanStack Table wrapper with default configuration
 */
export const useDataTable = <TData>({
  data,
  columns,
  getRowId,
  enableSorting = true,
  enableRowSelection = false,
  sorting: controlledSorting,
  onSortingChange: controlledOnSortingChange,
  rowSelection: controlledRowSelection,
  onRowSelectionChange: controlledOnRowSelectionChange,
}: UseDataTableProps<TData>) => {
  // Internal state for uncontrolled mode
  const [internalSorting, setInternalSorting] = useState<SortingState>([]);
  const [internalRowSelection, setInternalRowSelection] = useState<RowSelectionState>({});

  // Use controlled state if provided, otherwise use internal state
  const sorting = controlledSorting ?? internalSorting;
  const onSortingChange = controlledOnSortingChange ?? setInternalSorting;
  const rowSelection = controlledRowSelection ?? internalRowSelection;
  const onRowSelectionChange = controlledOnRowSelectionChange ?? setInternalRowSelection;

  // Memoize columns to prevent unnecessary re-renders
  const memoizedColumns = useMemo(() => columns, [columns]);

  // Create table instance
  const table = useReactTable({
    data,
    columns: memoizedColumns,
    getRowId,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: enableSorting ? getSortedRowModel() : undefined,

    // Sorting state
    state: {
      sorting,
      rowSelection,
    },
    onSortingChange,
    onRowSelectionChange,

    // Enable features
    enableSorting,
    enableRowSelection,
    enableSortingRemoval: true, // Enable 3-state sorting (null → asc → desc → null)
    enableMultiSort: false, // Single column sort only
  });

  return table;
};

"use client";
import type {
  ColumnOrderState,
  ColumnSizingState,
  OnChangeFn,
  VisibilityState,
} from "@tanstack/react-table";
import { useCallback, useState } from "react";
import { DEBOUNCE_DELAYS } from "../constants";
import type { TableStorageState } from "../types";
import { debounce, loadTableState, saveTableState } from "../utils";

interface UseTableStateProps {
  persistenceKey?: string;
  persistColumnOrder?: boolean;
  persistColumnSizing?: boolean;
  persistColumnVisibility?: boolean;
  initialColumnOrder?: ColumnOrderState;
  initialColumnSizing?: ColumnSizingState;
  initialColumnVisibility?: VisibilityState;
}

interface UseTableStateReturn {
  columnOrder: ColumnOrderState;
  setColumnOrder: OnChangeFn<ColumnOrderState>;
  columnSizing: ColumnSizingState;
  setColumnSizing: OnChangeFn<ColumnSizingState>;
  columnVisibility: VisibilityState;
  setColumnVisibility: OnChangeFn<VisibilityState>;
  resetState: () => void;
}

export function useTableState(props: UseTableStateProps): UseTableStateReturn {
  const {
    persistenceKey,
    persistColumnOrder = false,
    persistColumnSizing = false,
    persistColumnVisibility = false,
    initialColumnOrder = [],
    initialColumnSizing = {},
    initialColumnVisibility = {},
  } = props;

  // Load persisted state on mount
  const [loadedState] = useState(() => {
    if (persistenceKey && typeof window !== "undefined") {
      return loadTableState(persistenceKey);
    }
    return {};
  });

  // Initialize state
  const [columnOrder, setColumnOrderState] = useState<ColumnOrderState>(
    persistColumnOrder && loadedState.columnOrder ? loadedState.columnOrder : initialColumnOrder,
  );

  const [columnSizing, setColumnSizingState] = useState<ColumnSizingState>(
    persistColumnSizing && loadedState.columnSizing
      ? loadedState.columnSizing
      : initialColumnSizing,
  );

  const [columnVisibility, setColumnVisibilityState] = useState<VisibilityState>(
    persistColumnVisibility && loadedState.columnVisibility
      ? loadedState.columnVisibility
      : initialColumnVisibility,
  );

  // Debounced save function
  const debouncedSave = useCallback(
    debounce((state: Partial<TableStorageState>) => {
      if (persistenceKey) {
        saveTableState(persistenceKey, state);
      }
    }, DEBOUNCE_DELAYS.STORAGE),
    [],
  );

  // Update column order
  const setColumnOrder: OnChangeFn<ColumnOrderState> = useCallback(
    (updater) => {
      const newOrder = typeof updater === "function" ? updater(columnOrder) : updater;
      setColumnOrderState(newOrder);
      if (persistColumnOrder) {
        debouncedSave({ columnOrder: newOrder });
      }
    },
    [columnOrder, persistColumnOrder, debouncedSave],
  );

  // Update column sizing
  const setColumnSizing: OnChangeFn<ColumnSizingState> = useCallback(
    (updater) => {
      const newSizing = typeof updater === "function" ? updater(columnSizing) : updater;
      setColumnSizingState(newSizing);
      if (persistColumnSizing) {
        debouncedSave({ columnSizing: newSizing });
      }
    },
    [columnSizing, persistColumnSizing, debouncedSave],
  );

  // Update column visibility
  const setColumnVisibility: OnChangeFn<VisibilityState> = useCallback(
    (updater) => {
      const newVisibility = typeof updater === "function" ? updater(columnVisibility) : updater;
      setColumnVisibilityState(newVisibility);
      if (persistColumnVisibility) {
        debouncedSave({ columnVisibility: newVisibility });
      }
    },
    [columnVisibility, persistColumnVisibility, debouncedSave],
  );

  // Reset to initial state
  const resetState = useCallback(() => {
    setColumnOrderState(initialColumnOrder);
    setColumnSizingState(initialColumnSizing);
    setColumnVisibilityState(initialColumnVisibility);

    if (persistenceKey) {
      saveTableState(persistenceKey, {
        columnOrder: initialColumnOrder,
        columnSizing: initialColumnSizing,
        columnVisibility: initialColumnVisibility,
      });
    }
  }, [initialColumnOrder, initialColumnSizing, initialColumnVisibility, persistenceKey]);

  return {
    columnOrder,
    setColumnOrder,
    columnSizing,
    setColumnSizing,
    columnVisibility,
    setColumnVisibility,
    resetState,
  };
}

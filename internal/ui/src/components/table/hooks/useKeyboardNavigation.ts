"use client";
import type { Table } from "@tanstack/react-table";
import { useCallback, useEffect, useRef, useState } from "react";
import { DEFAULT_KEYBOARD_CONFIG, KEYBOARD_KEYS } from "../constants";
import type { KeyboardAction, KeyboardConfig } from "../types";

interface UseKeyboardNavigationProps<TData> {
  table: Table<TData>;
  config?: KeyboardConfig;
  enabled?: boolean;
  onRowClick?: (row: TData) => void;
}

export function useKeyboardNavigation<TData>(props: UseKeyboardNavigationProps<TData>) {
  const { table, config = DEFAULT_KEYBOARD_CONFIG, enabled = true, onRowClick } = props;

  const [focusedRowIndex, setFocusedRowIndex] = useState<number>(-1);
  const containerRef = useRef<HTMLDivElement>(null);

  const getAction = useCallback(
    (key: string): KeyboardAction | null => {
      const keyMap = { ...DEFAULT_KEYBOARD_CONFIG.customKeyMap, ...config.customKeyMap };
      return keyMap[key as keyof typeof keyMap] ?? null;
    },
    [config.customKeyMap],
  );

  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      if (!enabled) {
        return;
      }

      const action = getAction(event.key);
      if (!action) {
        return;
      }

      const rows = table.getRowModel().rows;
      if (rows.length === 0) {
        return;
      }

      // Prevent default browser behavior for table navigation keys
      const navigationKeys = [
        KEYBOARD_KEYS.ARROW_UP,
        KEYBOARD_KEYS.ARROW_DOWN,
        KEYBOARD_KEYS.SPACE,
        KEYBOARD_KEYS.ENTER,
      ] as const;

      if (navigationKeys.includes(event.key as (typeof navigationKeys)[number])) {
        event.preventDefault();
      }

      switch (action) {
        case "moveUp":
        case "moveDown": {
          const direction = action === "moveUp" ? -1 : 1;
          const newIndex = Math.max(0, Math.min(rows.length - 1, focusedRowIndex + direction));
          setFocusedRowIndex(newIndex);

          // Scroll into view
          const rowElement = containerRef.current?.querySelector(`[data-row-index="${newIndex}"]`);
          if (rowElement) {
            rowElement.scrollIntoView({ block: "nearest", behavior: "smooth" });
          }
          break;
        }

        case "selectRow": {
          if (focusedRowIndex >= 0 && focusedRowIndex < rows.length) {
            const row = rows[focusedRowIndex];
            if (row && onRowClick) {
              onRowClick(row.original);
            }
          }
          break;
        }

        case "toggleSelection": {
          if (focusedRowIndex >= 0 && focusedRowIndex < rows.length) {
            const row = rows[focusedRowIndex];
            if (row) {
              row.toggleSelected();
            }
          }
          break;
        }

        case "selectAll": {
          table.toggleAllRowsSelected(true);
          break;
        }

        case "clearSelection": {
          table.toggleAllRowsSelected(false);
          setFocusedRowIndex(-1);
          break;
        }

        case "escape": {
          setFocusedRowIndex(-1);
          table.toggleAllRowsSelected(false);
          // Remove focus from table
          if (document.activeElement instanceof HTMLElement) {
            document.activeElement.blur();
          }
          break;
        }

        default:
          break;
      }

      // Call custom callback if provided
      if (config.onKeyAction) {
        config.onKeyAction(action, event);
      }
    },
    [enabled, getAction, table, focusedRowIndex, onRowClick, config],
  );

  // Attach keyboard event listener
  useEffect(() => {
    if (!enabled) {
      return;
    }

    const container = containerRef.current;
    if (!container) {
      return;
    }

    container.addEventListener("keydown", handleKeyDown);
    return () => {
      container.removeEventListener("keydown", handleKeyDown);
    };
  }, [enabled, handleKeyDown]);

  return {
    containerRef,
    focusedRowIndex,
    setFocusedRowIndex,
  };
}

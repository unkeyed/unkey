import { STORAGE_KEYS } from "../constants";
import type { StorageConfig, TableStorageState } from "../types";

/**
 * Get storage instance (defaults to localStorage)
 */
function getStorage(config?: StorageConfig): Storage | null {
  return config?.storage ?? (typeof window !== "undefined" ? window.localStorage : null);
}

/**
 * Build storage key with prefix
 */
function buildStorageKey(persistenceKey: string, key: string): string {
  return `unkey-table-${persistenceKey}-${key}`;
}

/**
 * Save table state to storage
 */
export function saveTableState(
  persistenceKey: string,
  state: Partial<TableStorageState>,
  config?: StorageConfig,
): void {
  const storage = getStorage(config);

  if (!storage) {
    return;
  }

  try {
    if (state.columnOrder !== undefined) {
      const key = buildStorageKey(persistenceKey, STORAGE_KEYS.columnOrder);
      storage.setItem(key, JSON.stringify(state.columnOrder));
    }

    if (state.columnSizing !== undefined) {
      const key = buildStorageKey(persistenceKey, STORAGE_KEYS.columnSizing);
      storage.setItem(key, JSON.stringify(state.columnSizing));
    }

    if (state.columnVisibility !== undefined) {
      const key = buildStorageKey(persistenceKey, STORAGE_KEYS.columnVisibility);
      storage.setItem(key, JSON.stringify(state.columnVisibility));
    }
  } catch (error) {
    console.error("Error saving table state to storage:", error);
  }
}

/**
 * Load table state from storage
 */
export function loadTableState(persistenceKey: string, config?: StorageConfig): TableStorageState {
  const storage = getStorage(config);
  const state: TableStorageState = {};

  if (!storage) {
    return state;
  }

  try {
    // Load column order
    const columnOrderKey = buildStorageKey(persistenceKey, STORAGE_KEYS.columnOrder);
    const columnOrderValue = storage.getItem(columnOrderKey);
    if (columnOrderValue) {
      state.columnOrder = JSON.parse(columnOrderValue);
    }

    // Load column sizing
    const columnSizingKey = buildStorageKey(persistenceKey, STORAGE_KEYS.columnSizing);
    const columnSizingValue = storage.getItem(columnSizingKey);
    if (columnSizingValue) {
      state.columnSizing = JSON.parse(columnSizingValue);
    }

    // Load column visibility
    const columnVisibilityKey = buildStorageKey(persistenceKey, STORAGE_KEYS.columnVisibility);
    const columnVisibilityValue = storage.getItem(columnVisibilityKey);
    if (columnVisibilityValue) {
      state.columnVisibility = JSON.parse(columnVisibilityValue);
    }
  } catch (error) {
    console.error("Error loading table state from storage:", error);
  }

  return state;
}

/**
 * Clear table state from storage
 */
export function clearTableState(persistenceKey: string, config?: StorageConfig): void {
  const storage = getStorage(config);

  if (!storage) {
    return;
  }

  try {
    storage.removeItem(buildStorageKey(persistenceKey, STORAGE_KEYS.columnOrder));
    storage.removeItem(buildStorageKey(persistenceKey, STORAGE_KEYS.columnSizing));
    storage.removeItem(buildStorageKey(persistenceKey, STORAGE_KEYS.columnVisibility));
  } catch (error) {
    console.error("Error clearing table state from storage:", error);
  }
}

/**
 * Debounce utility for storage operations
 */
export function debounce<TArgs extends unknown[], TReturn>(
  func: (...args: TArgs) => TReturn,
  wait: number,
): (...args: TArgs) => void {
  let timeout: NodeJS.Timeout | null = null;

  return (...args: TArgs) => {
    if (timeout) {
      clearTimeout(timeout);
    }
    timeout = setTimeout(() => {
      func(...args);
    }, wait);
  };
}

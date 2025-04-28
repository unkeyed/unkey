"use client";
import { useCallback } from "react";
import { type FieldValues, type UseFormProps, type UseFormReturn, useForm } from "react-hook-form";

export type StorageType = "memory" | "session" | "local";

export type UsePersistedFormReturn<TFormValues extends FieldValues> = UseFormReturn<TFormValues> & {
  clearPersistedData: () => void;
  saveCurrentValues: () => void;
  loadSavedValues: () => Promise<boolean>;
};

// Create an in-memory storage singleton
const memoryStorage = new Map<string, string>();

/**
 * A React hook that extends `useForm` with persistent storage for form data.
 *
 * Persists form state to memory, sessionStorage, or localStorage, allowing form data to be saved, loaded, or cleared across sessions or reloads.
 *
 * @param storageKey - Unique key used to identify and store form data.
 * @param formOptions - Options passed to the underlying `useForm` hook.
 * @param storageType - Storage medium for persistence: `"memory"`, `"session"`, or `"local"`. Defaults to `"session"`.
 * @returns All standard `useForm` methods, plus `clearPersistedData`, `saveCurrentValues`, and `loadSavedValues` for managing persisted form state.
 */
export function usePersistedForm<TFormValues extends Record<string, any>>(
  storageKey: string,
  formOptions: UseFormProps<TFormValues>,
  storageType: StorageType = "session",
): UsePersistedFormReturn<TFormValues> {
  const methods = useForm<TFormValues>(formOptions);
  const { reset, getValues } = methods;

  // Helper to get the appropriate storage based on type
  const getStorage = useCallback(() => {
    switch (storageType) {
      case "memory":
        return {
          getItem: (key: string) => memoryStorage.get(key) || null,
          setItem: (key: string, value: string) => memoryStorage.set(key, value),
          removeItem: (key: string) => memoryStorage.delete(key),
        };
      case "local":
        return localStorage;
      default:
        return sessionStorage;
    }
  }, [storageType]);

  const loadSavedValues = useCallback(async () => {
    try {
      const storage = getStorage();
      const savedState = storage.getItem(storageKey);

      if (savedState) {
        const parsedState = JSON.parse(savedState);
        reset(parsedState);
        return true;
      }
    } catch (error) {
      console.error(`Error loading saved form state from ${storageType} storage:`, error);
    }
    return false;
  }, [reset, storageKey, getStorage, storageType]);

  // Save current form values
  const saveCurrentValues = useCallback(() => {
    try {
      const storage = getStorage();
      const currentValues = getValues();
      storage.setItem(storageKey, JSON.stringify(currentValues));
    } catch (error) {
      console.error(`Error saving form state to ${storageType} storage:`, error);
    }
  }, [getValues, storageKey, getStorage, storageType]);

  // Clear persisted data
  const clearPersistedData = useCallback(() => {
    try {
      const storage = getStorage();
      storage.removeItem(storageKey);
    } catch (error) {
      console.error(`Error clearing form state from ${storageType} storage:`, error);
    }
  }, [storageKey, getStorage, storageType]);

  // Return the methods with our added functions
  return {
    ...methods,
    clearPersistedData,
    saveCurrentValues,
    loadSavedValues,
  };
}

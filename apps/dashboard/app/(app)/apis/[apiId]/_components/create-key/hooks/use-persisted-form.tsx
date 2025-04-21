"use client";
import { useCallback } from "react";
import { type FieldValues, type UseFormProps, type UseFormReturn, useForm } from "react-hook-form";

export type UsePersistedFormReturn<TFormValues extends FieldValues> = UseFormReturn<TFormValues> & {
  clearPersistedData: () => void;
  saveCurrentValues: () => void;
  loadSavedValues: () => Promise<boolean>;
};

/**
 * Custom hook that extends useForm with session storage persistence
 */
export function usePersistedForm<TFormValues extends Record<string, any>>(
  storageKey: string,
  formOptions: UseFormProps<TFormValues>,
): UsePersistedFormReturn<TFormValues> {
  const methods = useForm<TFormValues>(formOptions);
  const { reset, getValues } = methods;

  const loadSavedValues = useCallback(async () => {
    try {
      const savedState = sessionStorage.getItem(storageKey);
      if (savedState) {
        const parsedState = JSON.parse(savedState);
        reset(parsedState);
        return true;
      }
    } catch (error) {
      console.error("Error loading saved form state:", error);
    }
    return false;
  }, [reset, storageKey]);

  // Save current form values
  const saveCurrentValues = useCallback(() => {
    try {
      const currentValues = getValues();
      sessionStorage.setItem(storageKey, JSON.stringify(currentValues));
    } catch (error) {
      console.error("Error saving form state:", error);
    }
  }, [getValues, storageKey]);

  // Clear persisted data
  const clearPersistedData = useCallback(() => {
    try {
      sessionStorage.removeItem(storageKey);
    } catch (error) {
      console.error("Error clearing form state:", error);
    }
  }, [storageKey]);

  // Return the methods with our added functions
  return {
    ...methods,
    clearPersistedData,
    saveCurrentValues,
    loadSavedValues,
  };
}

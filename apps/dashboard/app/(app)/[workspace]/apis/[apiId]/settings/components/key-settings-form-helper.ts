import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "@unkey/ui";
import type { z } from "zod";

const createInvalidationHelper = () => {
  const utils = trpc.useUtils();
  return {
    /**
     * Invalidates common API-related queries
     * Used after most API mutations (update name, whitelist, default bytes, etc.)
     */
    invalidateApiQueries: () => {
      utils.api.overview.query.invalidate();
      utils.api.queryApiKeyDetails.invalidate();
    },
  };
};

/**
 * Standard error handler for API mutations
 * Logs error and shows toast with the error message
 */
export const handleMutationError = (err: unknown) => {
  const error = err as { data: { code: string }; message: string };

  if (error.data?.code === "NOT_FOUND") {
    return "Resource not found. Please refresh and try again.";
  }

  if (error.data?.code === "UNAUTHORIZED") {
    return "You don't have permission to perform this action.";
  }

  if (error.data?.code === "BAD_REQUEST") {
    return error.message || "Invalid request. Please check your input.";
  }

  if (error.data?.code === "INTERNAL_SERVER_ERROR") {
    return "Server error occurred. Please try again later.";
  }

  return error.message || "An unexpected error occurred.";
};

/**
 * Standard mutation handlers for API operations
 */
export const createMutationHandlers = () => {
  const { invalidateApiQueries } = createInvalidationHelper();

  return {
    /**
     * Success handler for basic API updates (name, prefix, bytes, etc.)
     */
    onUpdateSuccess: (message: string) => () => {
      toast.success(message);
      invalidateApiQueries();
    },

    /**
     * Standard error handler with toast
     */
    onError: (err: unknown) => {
      toast.error(handleMutationError(err));
    },

    /**
     * Success handler for delete protection toggle
     */
    onDeleteProtectionSuccess: (apiName: string, enabled: boolean) => () => {
      toast.success(
        `Delete protection for ${apiName} has been ${enabled ? "enabled" : "disabled"}`,
      );
      invalidateApiQueries();
    },

    /**
     * Success handler for API deletion
     */
    onDeleteSuccess: (keyCount: number) => () => {
      toast.success("API Deleted", {
        description: `Your API and ${keyCount} keys have been deleted.`,
      });
      invalidateApiQueries();
    },
  };
};

/**
 * Common form configuration for API settings
 */
export const createApiFormConfig = <T extends z.ZodType>(schema: T) => {
  return {
    resolver: zodResolver(schema),
    mode: "all" as const,
    shouldFocusError: true,
    delayError: 100,
  };
};

/**
 * Standard form validation for unchanged values
 */
export const validateFormChange = <T>(
  currentValue: T,
  newValue: T,
  errorMessage: string,
): boolean => {
  if (currentValue === newValue || !newValue) {
    toast.error(errorMessage);
    return false;
  }
  return true;
};

/**
 * Standard button props for API setting forms
 */
export const getStandardButtonProps = (
  isValid: boolean,
  isSubmitting: boolean,
  isDirty: boolean,
) => ({
  size: "lg" as const,
  variant: "primary" as const,
  className: "h-full px-3.5 rounded-lg",
  disabled: !isValid || isSubmitting || !isDirty,
  type: "submit" as const,
  loading: isSubmitting,
  children: "Save",
});

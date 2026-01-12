/**
 * Sentry Error Filtering Logic
 *
 * This module provides centralized error filtering logic for Sentry configurations
 * across client, server, and edge runtimes. It implements the beforeSend hook
 * to filter expected tRPC errors while preserving original errors for client handling.
 *
 */

import type * as Sentry from "@sentry/nextjs";
import {
  createLogContext,
  generateRequestId,
  logOperation,
  logTRPCError,
} from "../logging/structured-logger";
import { classifyError } from "../utils/error-classification";

/**
 * Type definitions for Sentry event processing
 */
export type BeforeSendHook = Parameters<typeof Sentry.init>[0]["beforeSend"];

/**
 * Configuration options for error filtering
 */
export interface ErrorFilterOptions {
  /**
   * Whether to log filtered errors using structured logging
   * @default true
   */
  logFilteredErrors?: boolean;

  /**
   * Additional error types to filter (beyond tRPC errors)
   */
  additionalFilters?: Array<(error: unknown) => boolean>;

  /**
   * Custom context provider for enhanced logging
   */
  contextProvider?: () => { userId?: string; workspaceId?: string; requestId?: string };
}

/**
 * Creates a beforeSend hook for Sentry that filters expected tRPC errors
 *
 * This function implements the core error filtering logic that:
 * 1. Identifies expected tRPC errors using error classification utilities
 * 2. Logs filtered errors using structured logging instead of error reporting
 * 3. Preserves original errors for client-side handling
 * 4. Allows unexpected errors to be reported normally
 *
 * @param options - Configuration options for the error filter
 * @returns A beforeSend hook function for Sentry configuration
 */
export function createErrorFilter(options: ErrorFilterOptions = {}): BeforeSendHook {
  const { logFilteredErrors = true, additionalFilters = [], contextProvider } = options;

  return (event, hint) => {
    const error = hint.originalException;

    // Handle missing exception
    if (!error) {
      return event;
    }

    // Classify the error to determine how to handle it
    const classification = classifyError(error);

    // Check if this error should be filtered by tRPC classification
    if (classification.isExpected && classification.errorInfo) {
      if (logFilteredErrors) {
        // Create context for structured logging
        const baseContext = contextProvider?.() || {};
        const context = createLogContext({
          ...baseContext,
          timestamp: Date.now(),
        });

        // Log the filtered error using structured logging
        logTRPCError(classification.errorInfo, context);
      }

      // Return null to prevent Sentry error reporting
      return null;
    }

    // Check additional custom filters
    for (const filter of additionalFilters) {
      if (filter(error)) {
        if (logFilteredErrors) {
          const baseContext = contextProvider?.() || {};
          const context = createLogContext({
            ...baseContext,
            timestamp: Date.now(),
          });

          logOperation(
            "info",
            "Error filtered by custom filter",
            {
              error_type: error instanceof Error ? error.constructor.name : typeof error,
              error_message: error instanceof Error ? error.message : "Unknown error",
            },
            context,
          );
        }

        return null;
      }
    }

    // For unexpected errors, preserve the original event for Sentry reporting
    // This ensures that genuine issues are still tracked and debugged
    return event;
  };
}

/**
 * Default error filter configuration for client-side usage
 *
 * Includes standard tRPC error filtering with client-specific context
 */
export function createClientErrorFilter(): BeforeSendHook {
  return createErrorFilter({
    logFilteredErrors: true,
    contextProvider: () => {
      // In a real implementation, this would extract context from:
      // - Clerk auth state
      // - URL parameters
      // - Local storage
      // - React context
      return {
        userId: undefined, // Would get from auth context
        workspaceId: undefined, // Would get from URL or context
        requestId: generateRequestId(),
      };
    },
  });
}

/**
 * Default error filter configuration for server-side usage
 *
 * Includes standard tRPC error filtering with server-specific context
 */
export function createServerErrorFilter(): BeforeSendHook {
  return createErrorFilter({
    logFilteredErrors: true,
    contextProvider: () => {
      // In a real implementation, this would extract context from:
      // - Request headers
      // - Session data
      // - Database queries
      return {
        userId: undefined, // Would get from request context
        workspaceId: undefined, // Would get from request context
        requestId: generateRequestId(),
      };
    },
  });
}

/**
 * Default error filter configuration for edge runtime usage
 *
 * Includes standard tRPC error filtering with edge-specific context
 */
export function createEdgeErrorFilter(): BeforeSendHook {
  return createErrorFilter({
    logFilteredErrors: true,
    contextProvider: () => {
      // Edge runtime has limited context available
      return {
        userId: undefined,
        workspaceId: undefined,
        requestId: generateRequestId(),
      };
    },
  });
}

/**
 * Utility function to check if an error should be reported to Sentry
 *
 * This can be used in application code to make decisions about error handling
 * without going through the full Sentry pipeline.
 *
 * @param error - The error to check
 * @returns true if the error should be reported, false if it should be filtered
 */
export function shouldReportError(error: unknown): boolean {
  const classification = classifyError(error);
  return classification.shouldReport;
}

/**
 * Utility function to manually log a filtered error
 *
 * This can be used when you want to log an error that would normally be filtered
 * but need to capture it for debugging purposes.
 *
 * @param error - The error to log
 * @param context - Additional context for the log
 */
export function logFilteredError(
  error: unknown,
  context?: Partial<{ userId: string; workspaceId: string; requestId: string }>,
): void {
  const classification = classifyError(error);

  if (classification.errorInfo) {
    const logContext = createLogContext({
      ...context,
      timestamp: Date.now(),
    });

    logTRPCError(classification.errorInfo, logContext);
  } else {
    const logContext = createLogContext({
      ...context,
      timestamp: Date.now(),
    });

    logOperation(
      "warn",
      "Unclassified error logged",
      {
        error_type: error instanceof Error ? error.constructor.name : typeof error,
        error_message: error instanceof Error ? error.message : "Unknown error",
      },
      logContext,
    );
  }
}

/**
 * Enhanced error context preservation
 *
 * This function ensures that when errors are filtered, their context is preserved
 * for potential client-side handling or debugging purposes.
 *
 * @param error - The original error
 * @param additionalContext - Additional context to preserve
 * @returns Enhanced error with preserved context
 */
export function preserveErrorContext<T extends Error>(
  error: T,
  additionalContext: Record<string, unknown> = {},
): T {
  // Skip if already has preserved context to avoid TypeError on redefinition
  if (hasPreservedContext(error)) {
    return error;
  }

  // Add additional context as non-enumerable properties to avoid serialization issues
  Object.defineProperty(error, "__sentryContext", {
    value: {
      filtered: true,
      timestamp: Date.now(),
      ...additionalContext,
    },
    writable: false,
    enumerable: false,
    configurable: false,
  });

  return error;
}

/**
 * Type guard to check if an error has preserved Sentry context
 */
export function hasPreservedContext(
  error: unknown,
): error is Error & { __sentryContext: Record<string, unknown> } {
  return error instanceof Error && "__sentryContext" in error;
}

/**
 * Extracts preserved context from an error
 *
 * @param error - The error to extract context from
 * @returns The preserved context or null if none exists
 */
export function getPreservedContext(error: unknown): Record<string, unknown> | null {
  if (hasPreservedContext(error)) {
    return error.__sentryContext;
  }
  return null;
}

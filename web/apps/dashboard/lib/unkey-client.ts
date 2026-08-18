"use client";

import { Unkey } from "@unkey/api";
import * as errors from "@unkey/api/models/errors";

const fallbackErrorMessage = "An unexpected error occurred. Please try again later.";
let client: Unkey | null = null;

export function getUnkeyClient(): Unkey {
  if (client) {
    return client;
  }

  client = new Unkey({
    serverURL: new URL("/proxy/", window.location.origin).toString(),
  });

  return client;
}

/**
 * Maps an SDK error to a toast title and description. `fallbackMessage` names
 * the failed operation, e.g. "Failed to Delete App", and is only used for
 * errors we don't classify.
 */
export function getErrorToast(
  error: unknown,
  fallbackMessage: string,
): { message: string; description: string } {
  const toast = errorToasts.find(({ matches }) => matches(error));
  if (!toast) {
    return { message: fallbackMessage, description: getErrorMessage(error) };
  }

  return {
    message: toast.message,
    // A fixed description means the API detail isn't actionable for the user.
    description: toast.description ?? getErrorMessage(error, toast.fallback),
  };
}

export function getErrorMessage(error: unknown, fallback = fallbackErrorMessage): string {
  return isApiError(error) ? error.error.detail || fallback : fallback;
}

// SDK error classes that carry an API error body with a user-facing `detail`.
const API_ERRORS = [
  errors.BadRequestErrorResponse,
  errors.UnauthorizedErrorResponse,
  errors.ForbiddenErrorResponse,
  errors.NotFoundErrorResponse,
  errors.ConflictErrorResponse,
  errors.GoneErrorResponse,
  errors.PreconditionFailedErrorResponse,
  errors.UnprocessableEntityErrorResponse,
  errors.TooManyRequestsErrorResponse,
  errors.InternalServerErrorResponse,
  errors.ServiceUnavailableErrorResponse,
] as const;

function isApiError(error: unknown): error is InstanceType<(typeof API_ERRORS)[number]> {
  return API_ERRORS.some((apiError) => error instanceof apiError);
}

// Ordered: protected_resource must be checked before any broader match on the
// same status. `description` pins the text; `fallback` only fills in when the
// API sent no detail.
const errorToasts: {
  matches: (error: unknown) => boolean;
  message: string;
  description?: string;
  fallback?: string;
}[] = [
  {
    matches: (error) =>
      error instanceof errors.PreconditionFailedErrorResponse &&
      error.error.type.endsWith("/protected_resource"),
    message: "Delete Protection Enabled",
    fallback: "Disable delete protection and try again.",
  },
  {
    matches: (error) => error instanceof errors.UnauthorizedErrorResponse,
    message: "Authentication Required",
    description: "Your session may have expired. Please refresh the page and try again.",
  },
  {
    matches: (error) => error instanceof errors.ForbiddenErrorResponse,
    message: "Permission Denied",
    fallback: "You don't have permission to perform this action.",
  },
  {
    matches: (error) => error instanceof errors.NotFoundErrorResponse,
    message: "Not Found",
    fallback: "Unable to find the resource. Refresh and try again.",
  },
  {
    matches: (error) => error instanceof errors.ConflictErrorResponse,
    message: "Already Exists",
  },
  {
    matches: (error) => error instanceof errors.TooManyRequestsErrorResponse,
    message: "Too Many Requests",
    fallback: "Please wait a moment and try again.",
  },
  {
    matches: (error) =>
      error instanceof errors.InternalServerErrorResponse ||
      error instanceof errors.ServiceUnavailableErrorResponse,
    message: "Server Error",
    description:
      "We encountered an issue on our end. Please try again later or contact support at support@unkey.com.",
  },
  {
    // Covers connection, timeout, and abort errors: the request never got a response.
    matches: (error) => error instanceof errors.HTTPClientError,
    message: "Connection Problem",
    description: "Check your internet connection and try again.",
  },
];

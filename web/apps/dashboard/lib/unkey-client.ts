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
 * Maps an SDK error to a toast title and description. The description comes
 * from the API's public error detail, so the title only needs to classify the
 * failure. `fallbackMessage` names the failed operation, e.g. "Failed to Delete App".
 */
export function getErrorToast(
  error: unknown,
  fallbackMessage: string,
): { message: string; description: string } {
  if (
    error instanceof errors.PreconditionFailedErrorResponse &&
    error.error.type.endsWith("/protected_resource")
  ) {
    return {
      message: "Delete Protection Enabled",
      description: getErrorMessage(error, "Disable delete protection and try again."),
    };
  }
  if (error instanceof errors.UnauthorizedErrorResponse) {
    return {
      message: "Authentication Required",
      description: "Your session may have expired. Please refresh the page and try again.",
    };
  }
  if (error instanceof errors.ForbiddenErrorResponse) {
    return {
      message: "Permission Denied",
      description: getErrorMessage(error, "You don't have permission to perform this action."),
    };
  }
  if (error instanceof errors.NotFoundErrorResponse) {
    return {
      message: "Not Found",
      description: getErrorMessage(error, "Unable to find the resource. Refresh and try again."),
    };
  }
  if (error instanceof errors.ConflictErrorResponse) {
    return {
      message: "Already Exists",
      description: getErrorMessage(error),
    };
  }
  if (error instanceof errors.TooManyRequestsErrorResponse) {
    return {
      message: "Too Many Requests",
      description: getErrorMessage(error, "Please wait a moment and try again."),
    };
  }
  if (
    error instanceof errors.InternalServerErrorResponse ||
    error instanceof errors.ServiceUnavailableErrorResponse
  ) {
    return {
      message: "Server Error",
      description:
        "We encountered an issue on our end. Please try again later or contact support at support@unkey.com.",
    };
  }
  return {
    message: fallbackMessage,
    description: getErrorMessage(error),
  };
}

export function getErrorMessage(error: unknown, fallback = fallbackErrorMessage): string {
  if (
    error instanceof errors.BadRequestErrorResponse ||
    error instanceof errors.UnauthorizedErrorResponse ||
    error instanceof errors.ForbiddenErrorResponse ||
    error instanceof errors.NotFoundErrorResponse ||
    error instanceof errors.ConflictErrorResponse ||
    error instanceof errors.GoneErrorResponse ||
    error instanceof errors.PreconditionFailedErrorResponse ||
    error instanceof errors.UnprocessableEntityErrorResponse ||
    error instanceof errors.TooManyRequestsErrorResponse ||
    error instanceof errors.InternalServerErrorResponse ||
    error instanceof errors.ServiceUnavailableErrorResponse
  ) {
    return error.error.detail || fallback;
  }

  return fallback;
}

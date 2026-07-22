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

"use client";

import { Unkey } from "@unkey/api";
import { UnkeyError } from "@unkey/api/models/errors";

let client: Unkey | null = null;
const fallbackErrorMessage = "An unexpected error occurred. Please try again later.";

export function getUnkeyClient(): Unkey {
  if (client) {
    return client;
  }

  client = new Unkey({
    rootKey: "",
    serverURL: new URL("/proxy", window.location.origin).toString(),
  });

  return client;
}

export function getErrorMessage(error: unknown, fallback = fallbackErrorMessage): string {
  if (error instanceof UnkeyError) {
    return error.message;
  }
  return fallback;
}

"use client";

import { Unkey } from "@unkey/api";

let cached: Unkey | null = null;

export function getUnkeyClient(): Unkey {
  if (cached) {
    return cached;
  }
  cached = new Unkey({
    rootKey: "",
    serverURL: `${window.location.origin}/proxy`,
  });
  return cached;
}

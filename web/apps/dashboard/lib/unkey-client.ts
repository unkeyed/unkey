"use client";

import { Unkey } from "@unkey/api";

let client: Unkey | null = null;

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

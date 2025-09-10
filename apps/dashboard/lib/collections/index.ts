"use client";

import { ratelimitNamespaces } from "./ratelimit_namespaces";
import { ratelimitOverrides } from "./ratelimit_overrides";


export const collection = {
  ratelimitNamespaces,
  ratelimitOverrides,
};

// resets all collections data and preloads new
export async function reset() {
  await Promise.all(
    Object.values(collection).map(async (c) => {
      await c.cleanup();
      await c.preload();
    }),
  );
}

"use client";

import { domains } from "./domains";
import { projects } from "./projects";
import { ratelimitNamespaces } from "./ratelimit_namespaces";
import { ratelimitOverrides } from "./ratelimit_overrides";

export const collection = {
  ratelimitNamespaces,
  ratelimitOverrides,
  projects,
  domains
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

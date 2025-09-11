"use client";

import { deployments } from "./deployments";
import { domains } from "./domains";
import { projects } from "./projects";
import { ratelimitNamespaces } from "./ratelimit_namespaces";
import { ratelimitOverrides } from "./ratelimit_overrides";
import { environments } from "./environments";


export type { Deployment } from "./deployments";
export type { Domain } from "./domains";
export type { Project } from "./projects";
export type { RatelimitNamespace } from "./ratelimit_namespaces";
export type { RatelimitOverride } from "./ratelimit_overrides";
export type { Environment } from "./environments";

export const collection = {
  ratelimitNamespaces,
  ratelimitOverrides,
  projects,
  domains,
  deployments,
  environments
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

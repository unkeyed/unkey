"use client";
import { customDomains } from "./deploy/custom-domains";
import { deployments } from "./deploy/deployments";
import { domains } from "./deploy/domains";
import { envVars } from "./deploy/env-vars";
import { environmentSettings } from "./deploy/environment-settings";
import { environments } from "./deploy/environments";
import { projects } from "./deploy/projects";
import { ratelimitNamespaces } from "./ratelimit/namespaces";
import { ratelimitOverrides } from "./ratelimit/overrides";

// Export types
export type { CustomDomain } from "./deploy/custom-domains";
export type { Deployment } from "./deploy/deployments";
export type { Domain } from "./deploy/domains";
export type { EnvVar } from "./deploy/env-vars";
export type { EnvironmentSettings } from "./deploy/environment-settings";
export type { Project } from "./deploy/projects";
export type { RatelimitNamespace } from "./ratelimit/namespaces";
export type { RatelimitOverride } from "./ratelimit/overrides";
export type { Environment } from "./deploy/environments";

// Global collections
export const collection = {
  projects,
  ratelimitNamespaces,
  ratelimitOverrides,
  environments,
  domains,
  deployments,
  customDomains,
  environmentSettings,
  envVars,
} as const;

export async function reset() {
  // Clean up all global collections
  await Promise.all(Object.values(collection).map((c) => c.cleanup()));
  // Preload global collections after cleanup
  await Promise.all(Object.values(collection).map((c) => c.preload()));
}

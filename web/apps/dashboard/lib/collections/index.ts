"use client";
import { customDomains } from "./deploy/custom-domains";
import { domains } from "./deploy/domains";
import { envVars } from "./deploy/env-vars";
import { environmentSettings } from "./deploy/environment-settings";
import { environments } from "./deploy/environments";
import { policies } from "./deploy/policies";
import { ratelimitOverrides } from "./ratelimit/overrides";

// Export types
export type { CustomDomain } from "./deploy/custom-domains";
export type { DeploymentStatus } from "./deploy/deployment-status";
export { DEPLOYMENT_STATUSES } from "./deploy/deployment-status";
export type { Deployment, DeploymentSummary } from "@/lib/deploy-data/deployment";
export type { Domain } from "./deploy/domains";
export type { EnvVar } from "./deploy/env-vars";
export type { EnvironmentSettings } from "./deploy/environment-settings";
export type { PolicyRow } from "./deploy/policies";
export type { KeyauthPolicy, Policy, PolicyType } from "./deploy/policies.schema";
export type { RatelimitOverride } from "./ratelimit/overrides";
export type { Environment } from "./deploy/environments";

// Global collections
export const collection = {
  ratelimitOverrides,
  environments,
  domains,
  customDomains,
  environmentSettings,
  envVars,
  policies,
} as const;

export async function reset() {
  // Clean up all global collections
  await Promise.all(Object.values(collection).map((c) => c.cleanup()));
  // Preload global collections after cleanup
  await Promise.all(Object.values(collection).map((c) => c.preload()));
}

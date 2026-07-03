"use client";
import { apps } from "./deploy/apps";
import { customDomains } from "./deploy/custom-domains";
import { deployments } from "./deploy/deployments";
import { domains } from "./deploy/domains";
import { envVars } from "./deploy/env-vars";
import { environmentSettings } from "./deploy/environment-settings";
import { environments } from "./deploy/environments";
import { projects } from "./deploy/projects";
import { sentinelPolicies } from "./deploy/sentinel-policies";
import { ratelimitNamespaces } from "./ratelimit/namespaces";
import { ratelimitOverrides } from "./ratelimit/overrides";
import { scheduledDeletions } from "./settings/scheduled-deletions";

// Export types
export type { App } from "./deploy/apps";
export type { CustomDomain } from "./deploy/custom-domains";
export type { DeploymentStatus } from "./deploy/deployment-status";
export { DEPLOYMENT_STATUSES } from "./deploy/deployment-status";
export type { Deployment } from "./deploy/deployments";
export type { Domain } from "./deploy/domains";
export type { EnvVar } from "./deploy/env-vars";
export type { EnvironmentSettings } from "./deploy/environment-settings";
export type { Project } from "./deploy/projects";
export type { SentinelPolicyRow } from "./deploy/sentinel-policies";
export type {
  KeyauthPolicy,
  SentinelConfig,
  SentinelPolicy,
  SentinelPolicyType,
} from "./deploy/sentinel-policies.schema";
export type { RatelimitNamespace } from "./ratelimit/namespaces";
export type { RatelimitOverride } from "./ratelimit/overrides";
export type { Environment } from "./deploy/environments";
export type { ScheduledDeletion } from "./settings/scheduled-deletions";

// Global collections
export const collection = {
  projects,
  apps,
  ratelimitNamespaces,
  ratelimitOverrides,
  environments,
  domains,
  deployments,
  customDomains,
  environmentSettings,
  envVars,
  sentinelPolicies,
  scheduledDeletions,
} as const;

export async function reset() {
  // Clean up all global collections
  await Promise.all(Object.values(collection).map((c) => c.cleanup()));
  // Preload global collections after cleanup
  await Promise.all(Object.values(collection).map((c) => c.preload()));
}

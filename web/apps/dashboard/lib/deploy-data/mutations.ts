"use client";

import { collection } from "@/lib/collections";
import type { App } from "@/lib/collections/deploy/apps";
import { retryDomainVerification } from "@/lib/collections/deploy/custom-domains";
import {
  type EnvVar,
  makeVariablesSensitive,
  saveVariables,
} from "@/lib/collections/deploy/env-vars";
import {
  type EnvironmentSettings,
  applyDefaultSettings,
} from "@/lib/collections/deploy/environment-settings";
import type { CreateProjectRequestSchema } from "@/lib/collections/deploy/projects";
import { SERVER_PLACEHOLDER } from "@/lib/collections/deploy/utils";
import { DuplicateKeyError } from "@tanstack/react-db";
import { z } from "zod";

type AppChanges = Partial<Pick<App, "name" | "defaultBranch" | "imageReference">>;

type EnvVarChanges = Pick<EnvVar, "key" | "value" | "description" | "type">;

// A project create that finds another create still in flight reports it instead of throwing
export type CreateProjectResult =
  | { status: "created"; projectId: string }
  | { status: "duplicate" };

export function useAppMutations() {
  return APP_MUTATIONS;
}

export function useCustomDomainMutations() {
  return CUSTOM_DOMAIN_MUTATIONS;
}

// Project writes toast their own progress and outcome
export function useProjectMutations() {
  return PROJECT_MUTATIONS;
}

// Env var writes toast their own outcome and refetch the app's variables
export function useEnvVarMutations() {
  return ENV_VAR_MUTATIONS;
}

export function useEnvironmentSettingsMutations() {
  return ENVIRONMENT_SETTINGS_MUTATIONS;
}

const APP_MUTATIONS = {
  updateApp: (appId: string, changes: AppChanges): Promise<unknown> =>
    collection.apps.update(appId, { optimistic: false }, (draft) => {
      Object.assign(draft, changes);
    }).isPersisted.promise,
  deleteApp: (appId: string): void => {
    collection.apps.delete(appId);
  },
};

const CUSTOM_DOMAIN_MUTATIONS = {
  addCustomDomain: (
    domain: { domain: string; projectId: string; appId: string; environmentId: string },
    workspaceSlug: string,
  ): Promise<unknown> =>
    collection.customDomains.insert(
      {
        ...domain,
        id: crypto.randomUUID(),
        verificationStatus: "pending",
        dnsRecords: [],
        verificationError: null,
        domainConnectProvider: null,
        domainConnectUrl: null,
        createdAt: Date.now(),
        updatedAt: null,
      },
      { metadata: { workspaceSlug } },
    ).isPersisted.promise,
  deleteCustomDomain: (customDomainId: string): void => {
    collection.customDomains.delete(customDomainId);
  },
  retryDomainVerification,
};

const PROJECT_MUTATIONS = {
  createProject: async (values: CreateProjectRequestSchema): Promise<CreateProjectResult> => {
    try {
      const tx = collection.projects.insert({
        name: values.name,
        slug: values.slug,
        isDefault: false,
        id: SERVER_PLACEHOLDER,
        createdAt: Date.now(),
      });
      await tx.isPersisted.promise;
      const { projectId } = createdProjectSchema.parse(tx.metadata);
      return { status: "created", projectId };
    } catch (error) {
      if (error instanceof DuplicateKeyError) {
        return { status: "duplicate" };
      }
      throw error;
    }
  },
  renameProject: (projectId: string, name: string): Promise<unknown> =>
    collection.projects.update(projectId, (draft) => {
      draft.name = name;
    }).isPersisted.promise,
  // Not optimistic: the project stays readable until the server has deleted it
  deleteProject: (projectId: string): Promise<unknown> =>
    collection.projects.delete(projectId, { optimistic: false }).isPersisted.promise,
};

const ENV_VAR_MUTATIONS = {
  updateEnvVar: (envVarId: string, changes: EnvVarChanges): Promise<unknown> =>
    collection.envVars.update(envVarId, { optimistic: false }, (draft) => {
      Object.assign(draft, changes);
    }).isPersisted.promise,
  deleteEnvVars: (envVarIds: string[]): void => {
    collection.envVars.delete(envVarIds);
  },
  saveVariables,
  makeVariablesSensitive,
};

const ENVIRONMENT_SETTINGS_MUTATIONS = {
  /**
   * One transaction for every environment: the collection refetches after each
   * transaction settles, so one per environment would multiply the reads.
   * Not optimistic by default, so a failed save leaves the row and the form's edits alone.
   */
  updateEnvironmentSettings: (
    environmentIds: string[],
    update: (draft: EnvironmentSettings) => void,
    { optimistic = false }: { optimistic?: boolean } = {},
  ): Promise<unknown> =>
    collection.environmentSettings.update(environmentIds, { optimistic }, (drafts) =>
      drafts.forEach(update),
    ).isPersisted.promise,
  applyDefaultSettings,
};

const createdProjectSchema = z.object({ projectId: z.string() });

"use client";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { parseLoadSubsetOptions, queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import type {
  Environment,
  V2EnvironmentsUpdateSettingsRequestBody,
} from "@unkey/api/models/components";
import { toast } from "@unkey/ui";
import { useSyncExternalStore } from "react";
import { z } from "zod";
import { queryClient } from "../client";
import { extractStringFilter } from "./utils";

const healthcheckSchema = z
  .object({
    method: z.enum(["GET", "POST"]),
    path: z.string(),
    intervalSeconds: z.number(),
    timeoutSeconds: z.number(),
    failureThreshold: z.number(),
    initialDelaySeconds: z.number(),
  })
  .nullable();

const schema = z.object({
  environmentId: z.string(),
  // The API identifies an environment by project, app, and environment
  // together, so a mutation needs all three.
  projectId: z.string(),
  appId: z.string(),
  // Build settings
  autoDeploy: z.boolean().default(true),
  dockerfile: z.string(),
  dockerContext: z.string(),
  // Empty means "let Railpack auto-detect". Overrides Railpack's build command
  // so monorepos can scope the build to a single app.
  buildCommand: z.string().default(""),
  watchPaths: z.array(z.string()).default([]),
  // Runtime settings
  port: z.number().int(),
  cpuMillicores: z.number().int(),
  memoryMib: z.number().int(),
  storageMib: z.number().int(),
  command: z.array(z.string()),
  healthcheck: healthcheckSchema,
  regions: z.array(
    z.object({
      name: z.string(),
      replicasMin: z.number().int().min(1),
      replicasMax: z.number().int().min(1),
    }),
  ),
  shutdownSignal: z.string(),
  upstreamProtocol: z.enum(["http1", "h2c"]).default("http1"),
  openapiSpecPath: z.string().nullable().default(null),
});

/**
 * Environment settings collection - flattened build + runtime settings.
 *
 * IMPORTANT: All queries MUST filter by projectId and appId. Add environmentId
 * to select one environment; the collection holds every environment of the app,
 * so that filter runs in memory and needs no extra request:
 * .where(({ s }) => and(
 *   eq(s.projectId, projectId),
 *   eq(s.appId, appId),
 *   eq(s.environmentId, environmentId),
 * ))
 */
export const environmentSettings = createCollection<EnvironmentSettings, string>(
  queryCollectionOptions({
    queryClient,
    queryKey: (opts) => {
      const { filters } = parseLoadSubsetOptions(opts);
      const appId = extractStringFilter(filters, "appId");
      return appId ? ["environmentSettings", appId] : ["environmentSettings"];
    },
    retry: 3,
    syncMode: "on-demand",
    queryFn: async (ctx) => {
      const { filters } = parseLoadSubsetOptions(ctx.meta?.loadSubsetOptions);
      const projectId = extractStringFilter(filters, "projectId");
      const appId = extractStringFilter(filters, "appId");

      if (!projectId || !appId) {
        throw new Error("Query must include eq(s.projectId, ...) and eq(s.appId, ...) constraints");
      }

      // One request carries the settings of every environment in the app, so
      // selecting one environment costs no extra round trip.
      const result = await getUnkeyClient().environments.listEnvironments({
        project: projectId,
        app: appId,
      });

      return result.data.map((env) => flattenEnvironment(projectId, appId, env));
    },
    getKey: (item) => item.environmentId,
    id: "environmentSettings",
    onUpdate: async ({ transaction }) => {
      const silent = transaction.metadata?.silent === true;
      // A transaction can carry one environment or every environment of an app,
      // so send them together and report the outcome once.
      await dispatchSettingsMutations(
        transaction.mutations.map((m) => ({ original: m.original, modified: m.modified })),
        silent,
      );
    },
  }),
);

export type EnvironmentSettings = z.infer<typeof schema>;

/** Default values for environment settings fields (excluding regions, which are runtime-dependent). */
export const ENVIRONMENT_SETTINGS_DEFAULTS = {
  autoDeploy: true,
  // Empty means "no Dockerfile configured" — the app is built with Railpack.
  dockerfile: "",
  dockerContext: ".",
  // Empty means "let Railpack auto-detect" the build command.
  buildCommand: "",
  port: 8080,
  cpuMillicores: 250,
  memoryMib: 256,
  storageMib: 0,
  shutdownSignal: "SIGTERM",
  upstreamProtocol: "http1",
} as const;

function changed<T>(a: T, b: T): boolean {
  return JSON.stringify(a) !== JSON.stringify(b);
}

function flattenEnvironment(
  projectId: string,
  appId: string,
  env: Environment,
): EnvironmentSettings {
  const d = ENVIRONMENT_SETTINGS_DEFAULTS;
  const { build, runtime } = env;
  return {
    environmentId: env.id,
    projectId,
    appId,
    autoDeploy: build?.autoDeploy ?? d.autoDeploy,
    dockerfile: build?.dockerfile ?? d.dockerfile,
    dockerContext: build?.rootDirectory || d.dockerContext,
    buildCommand: build?.buildCommand ?? d.buildCommand,
    watchPaths: build?.watchPaths ?? [],
    port: runtime?.port ?? d.port,
    // The API uses vCPUs. The UI uses millicores.
    cpuMillicores: runtime ? Math.round(runtime.vCpus * 1000) : d.cpuMillicores,
    memoryMib: runtime?.memoryMib ?? d.memoryMib,
    storageMib: runtime?.storageMib ?? d.storageMib,
    command: runtime?.command ?? [],
    healthcheck: runtime?.healthcheck
      ? {
          method: runtime.healthcheck.method,
          path: runtime.healthcheck.path,
          intervalSeconds: runtime.healthcheck.intervalSeconds ?? 10,
          timeoutSeconds: runtime.healthcheck.timeoutSeconds ?? 5,
          failureThreshold: runtime.healthcheck.failureThreshold ?? 3,
          initialDelaySeconds: runtime.healthcheck.initialDelaySeconds ?? 0,
        }
      : null,
    regions: (env.regions ?? []).map((r) => ({
      name: r.name,
      replicasMin: r.replicas.min,
      replicasMax: r.replicas.max,
    })),
    shutdownSignal: runtime?.shutdownSignal ?? d.shutdownSignal,
    upstreamProtocol: runtime?.upstreamProtocol ?? d.upstreamProtocol,
    openapiSpecPath: runtime?.openapiSpecPath ?? null,
  };
}

/**
 * Makes the updateSettings request body for the fields that changed between
 * `original` and `modified`. The API keeps a field that the body does not
 * contain. To clear a nullable field, send null.
 *
 * Returns null if nothing changed, so the caller can skip the request.
 *
 * This function is pure. It shows no toasts and sends no requests.
 */
export function buildSettingsUpdate(
  original: EnvironmentSettings,
  modified: EnvironmentSettings,
): V2EnvironmentsUpdateSettingsRequestBody | null {
  const body: V2EnvironmentsUpdateSettingsRequestBody = {
    project: original.projectId,
    app: original.appId,
    environment: original.environmentId,
  };
  let dirty = false;

  if (modified.autoDeploy !== original.autoDeploy) {
    body.autoDeploy = modified.autoDeploy;
    dirty = true;
  }

  if (modified.dockerfile !== original.dockerfile) {
    // An empty string means no Dockerfile. The API clears the field with null.
    body.dockerfile = modified.dockerfile || null;
    dirty = true;
  }

  if (modified.dockerContext !== original.dockerContext) {
    body.rootDirectory = modified.dockerContext;
    dirty = true;
  }

  if (modified.buildCommand !== original.buildCommand) {
    // An empty string lets Railpack find the command. The API spells that null.
    body.buildCommand = modified.buildCommand || null;
    dirty = true;
  }

  if (changed(original.watchPaths, modified.watchPaths)) {
    body.watchPaths = modified.watchPaths;
    dirty = true;
  }

  if (modified.port !== original.port) {
    body.port = modified.port;
    dirty = true;
  }

  if (modified.cpuMillicores !== original.cpuMillicores) {
    body.vCpus = modified.cpuMillicores / 1000;
    dirty = true;
  }

  if (modified.memoryMib !== original.memoryMib) {
    body.memoryMib = modified.memoryMib;
    dirty = true;
  }

  if (modified.storageMib !== original.storageMib) {
    body.storageMib = modified.storageMib;
    dirty = true;
  }

  if (changed(original.command, modified.command)) {
    body.command = modified.command;
    dirty = true;
  }

  if (changed(original.healthcheck, modified.healthcheck)) {
    body.healthcheck = modified.healthcheck;
    dirty = true;
  }

  if (modified.upstreamProtocol !== original.upstreamProtocol) {
    body.upstreamProtocol = modified.upstreamProtocol;
    dirty = true;
  }

  if (modified.openapiSpecPath !== original.openapiSpecPath) {
    body.openapiSpecPath = modified.openapiSpecPath;
    dirty = true;
  }

  // The API rejects an empty list, because an environment must have one region
  // or more.
  if (modified.regions.length > 0 && changed(original.regions, modified.regions)) {
    body.regions = modified.regions.map((r) => ({
      name: r.name,
      replicas: { min: r.replicasMin, max: r.replicasMax },
    }));
    dirty = true;
  }

  return dirty ? body : null;
}

/**
 * Persist real defaults for a new environment at create time, so a user who
 * bails before configuring deployment still has usable settings instead of the
 * empty placeholders it starts with.
 */
export function applyDefaultSettings(
  projectId: string,
  appId: string,
  environmentId: string,
  regionNames: string[],
): Promise<unknown> {
  const d = ENVIRONMENT_SETTINGS_DEFAULTS;

  return getUnkeyClient().environments.updateSettings({
    project: projectId,
    app: appId,
    environment: environmentId,
    autoDeploy: d.autoDeploy,
    dockerfile: null,
    rootDirectory: d.dockerContext,
    buildCommand: null,
    watchPaths: [],
    port: d.port,
    vCpus: d.cpuMillicores / 1000,
    memoryMib: d.memoryMib,
    storageMib: d.storageMib,
    command: [],
    healthcheck: null,
    upstreamProtocol: d.upstreamProtocol,
    openapiSpecPath: null,
    // The API rejects an empty list, so keep the regions if the available ones
    // have not loaded.
    ...(regionNames.length > 0
      ? { regions: regionNames.map((name) => ({ name, replicas: { min: 1, max: 1 } })) }
      : {}),
  });
}

async function dispatchSettingsMutations(
  changes: { original: EnvironmentSettings; modified: EnvironmentSettings }[],
  silent = false,
): Promise<void> {
  const bodies = changes
    .map(({ original, modified }) => buildSettingsUpdate(original, modified))
    .filter((body) => body !== null);

  if (bodies.length === 0) {
    return;
  }

  const client = getUnkeyClient();
  const mutation = Promise.all(bodies.map((body) => client.environments.updateSettings(body)));

  if (!silent) {
    toast.promise(mutation, {
      loading: "Saving settings...",
      success: "Settings updated",
      error: (err) => ({
        message: "Failed to update settings",
        description: getErrorMessage(err),
      }),
    });
  }
  await trackSave(mutation);
}

/**
 * Store for tracking in-flight and completed collection saves.
 *
 * Shared by environment-settings and env-vars collections so the
 * pending-redeploy banner reacts to mutations from either source.
 */
const saveStore = {
  pendingSaves: 0,
  savedCount: 0,
  dismissedAtCount: 0,
  listeners: new Set<() => void>(),
  notify() {
    for (const cb of this.listeners) {
      cb();
    }
  },
  subscribe(cb: () => void): () => void {
    this.listeners.add(cb);
    return () => {
      this.listeners.delete(cb);
    };
  },
  dismiss() {
    this.dismissedAtCount = this.savedCount;
    this.notify();
  },
};

export function trackSave<T>(promise: Promise<T>): Promise<T> {
  saveStore.pendingSaves++;
  saveStore.notify();
  return promise.then(
    (result) => {
      saveStore.savedCount++;
      saveStore.pendingSaves--;
      saveStore.notify();
      return result;
    },
    (err) => {
      saveStore.pendingSaves--;
      saveStore.notify();
      throw err;
    },
  );
}

export function useSettingsIsSaving(): boolean {
  return useSyncExternalStore(
    (cb) => saveStore.subscribe(cb),
    () => saveStore.pendingSaves > 0,
  );
}

/** Returns true when there are saves the user hasn't dismissed yet. Survives navigation. */
export function useSettingsBannerVisible(): boolean {
  return useSyncExternalStore(
    (cb) => saveStore.subscribe(cb),
    () => saveStore.savedCount > saveStore.dismissedAtCount,
  );
}

/** Dismisses the pending-redeploy banner until a new save occurs. */
export function dismissSettingsBanner(): void {
  saveStore.dismiss();
}

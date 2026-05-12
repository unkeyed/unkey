/**
 * Axiom log-drain extension provider.
 *
 * The "axiom" extension is the first live entry in the marketplace; installing
 * it provisions a real `log_drains` row (and an encrypted `log_drain_credentials`
 * row) under the hood. Users still configure the drain via the marketplace
 * install wizard — the standalone log-drain CRUD has been removed in favor of
 * this hook chain.
 *
 * The provider deliberately owns the `log_drains` lifecycle. The extension
 * router stays generic and never references the log_drains schema directly;
 * any cascade behavior (soft-delete on uninstall, enable/disable, config
 * updates) is implemented here so a non-logdrain extension can land later
 * without touching this file.
 */
import { VaultService } from "@/gen/proto/vault/v1/service_pb";
import { and, db, eq, isNull, schema } from "@/lib/db";
import { createVaultClient } from "@/lib/vault-client";
import type { LogDrainEnvironment, LogDrainFilters, LogDrainSource } from "@unkey/db/src/schema";
import { newId } from "@unkey/id";
import type { LiveProvider } from "../providers";

const vault = createVaultClient(VaultService);

type AxiomConfig = {
  dataset: string;
  endpoint?: string;
  token?: string;
  sources: LogDrainSource[];
  environments: LogDrainEnvironment[];
  minSeverity?: string;
  includeBodies?: boolean;
  excludePaths?: string[];
};

/** Narrow the freeform installation `config` JSON into the Axiom shape. */
function parseAxiomConfig(raw: Record<string, string | boolean | string[]>): AxiomConfig {
  const dataset = typeof raw.dataset === "string" ? raw.dataset.trim() : "";
  if (dataset.length === 0) {
    throw new Error("axiom extension config: dataset is required");
  }

  const endpointRaw = typeof raw.endpoint === "string" ? raw.endpoint.trim() : "";
  const tokenRaw = typeof raw.token === "string" ? raw.token : "";

  const sources = Array.isArray(raw.sources)
    ? (raw.sources.filter((s) => s === "runtime" || s === "request") as LogDrainSource[])
    : [];
  if (sources.length === 0) {
    throw new Error("axiom extension config: at least one source is required");
  }

  const environments = Array.isArray(raw.environments)
    ? (raw.environments.filter(
        (e) => e === "production" || e === "preview",
      ) as LogDrainEnvironment[])
    : [];
  if (environments.length === 0) {
    throw new Error("axiom extension config: at least one environment is required");
  }

  return {
    dataset,
    endpoint: endpointRaw.length > 0 ? endpointRaw : undefined,
    token: tokenRaw.length > 0 ? tokenRaw : undefined,
    sources,
    environments,
    minSeverity: typeof raw.minSeverity === "string" ? raw.minSeverity : undefined,
    includeBodies: typeof raw.includeBodies === "boolean" ? raw.includeBodies : undefined,
    excludePaths: Array.isArray(raw.excludePaths)
      ? raw.excludePaths.filter((p): p is string => typeof p === "string")
      : undefined,
  };
}

/**
 * Project the install-wizard config onto the `log_drains` filters JSON column.
 * Only emits a sub-object when the relevant source is selected so the runtime
 * does not see filters for sources it isn't asked to forward.
 */
function buildFilters(config: AxiomConfig): LogDrainFilters {
  const filters: LogDrainFilters = {};

  if (config.sources.includes("runtime") && config.minSeverity && config.minSeverity !== "all") {
    filters.runtime = {
      minSeverity: config.minSeverity as "debug" | "info" | "warn" | "error",
    };
  }

  if (config.sources.includes("request")) {
    const request: NonNullable<LogDrainFilters["request"]> = {};
    if (config.includeBodies !== undefined) {
      request.includeBodies = config.includeBodies;
    }
    if (config.excludePaths && config.excludePaths.length > 0) {
      request.excludePaths = config.excludePaths;
    }
    if (Object.keys(request).length > 0) {
      filters.request = request;
    }
  }

  return filters;
}

/** Locate the log_drains row that backs an installation, if any. */
async function findDrainForInstallation(installationId: string) {
  return db.query.logDrains.findFirst({
    where: and(
      eq(schema.logDrains.extensionInstallationId, installationId),
      isNull(schema.logDrains.deletedAt),
    ),
    columns: { id: true, workspaceId: true },
  });
}

export const logDrainProvider: LiveProvider = {
  async onInstall(ctx, installation) {
    const config = parseAxiomConfig(installation.config);
    if (!config.token) {
      throw new Error("axiom extension config: token is required on install");
    }

    const drainId = newId("logDrain");
    const now = Date.now();

    // Encrypt the pasted token via Vault keyed on the workspace, matching
    // env-vars and ACME certificates. Re-encrypts when KMS material rotates.
    const { encrypted, keyId } = await vault.encrypt({
      keyring: ctx.workspaceId,
      data: config.token,
    });

    // Atomic insert across log_drains + log_drain_credentials. State and
    // cursor rows are bootstrapped lazily by the coordinator on first tick.
    await db.transaction(async (tx) => {
      await tx.insert(schema.logDrains).values({
        id: drainId,
        workspaceId: ctx.workspaceId,
        projectId: installation.projectId,
        name: installation.instanceName,
        provider: "axiom",
        config: {
          dataset: config.dataset,
          ...(config.endpoint ? { endpoint: config.endpoint } : {}),
        },
        sources: config.sources,
        environments: config.environments,
        apps: [],
        filters: buildFilters(config),
        deliveryMode: "batch",
        enabled: installation.status !== "disabled",
        extensionInstallationId: installation.id,
        createdAt: now,
      });

      await tx.insert(schema.logDrainCredentials).values({
        drainId,
        source: "paste",
        encryptedCredentials: encrypted,
        encryptionKeyId: keyId,
        oauthGrantId: null,
        updatedAt: now,
      });
    });
  },

  async onUpdate(ctx, installation, patch) {
    const drain = await findDrainForInstallation(installation.id);
    if (!drain) {
      // Installation has no backing drain (older install pre-provider, or a
      // drain that was hard-deleted). Treat as a no-op rather than synthesizing
      // a new row — the user can uninstall + reinstall to recover.
      return;
    }

    const nextRaw = patch.config ?? installation.config;
    const next = parseAxiomConfig(nextRaw);
    const now = Date.now();

    await db
      .update(schema.logDrains)
      .set({
        name: patch.instanceName ?? installation.instanceName,
        config: {
          dataset: next.dataset,
          ...(next.endpoint ? { endpoint: next.endpoint } : {}),
        },
        sources: next.sources,
        environments: next.environments,
        filters: buildFilters(next),
        updatedAt: now,
      })
      .where(eq(schema.logDrains.id, drain.id));

    // Token rotation: only when the form supplies a non-empty value, so a
    // user can edit metadata without re-pasting their token (the secret
    // field re-renders blank).
    if (next.token) {
      const { encrypted, keyId } = await vault.encrypt({
        keyring: ctx.workspaceId,
        data: next.token,
      });

      await db
        .update(schema.logDrainCredentials)
        .set({
          source: "paste",
          encryptedCredentials: encrypted,
          encryptionKeyId: keyId,
          oauthGrantId: null,
        })
        .where(eq(schema.logDrainCredentials.drainId, drain.id));
    }
  },

  async onSetEnabled(_ctx, installation, enabled) {
    const drain = await findDrainForInstallation(installation.id);
    if (!drain) {
      return;
    }
    await db
      .update(schema.logDrains)
      .set({ enabled, updatedAt: Date.now() })
      .where(eq(schema.logDrains.id, drain.id));
  },

  async onUninstall(_ctx, installation) {
    const drain = await findDrainForInstallation(installation.id);
    if (!drain) {
      return;
    }
    // The provider owns this runtime row, so a soft-delete here is the
    // provider's own choice, not a generic cascade. Coordinator filters
    // on `deleted_at IS NULL` so the next tick stops emitting.
    const now = Date.now();
    await db
      .update(schema.logDrains)
      .set({ enabled: false, deletedAt: now, updatedAt: now })
      .where(eq(schema.logDrains.id, drain.id));
  },
};

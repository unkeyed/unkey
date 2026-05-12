/**
 * Installation store with two backends merged behind one hook.
 *
 * - "preview" extensions persist in `localStorage` so the marketplace demo
 *   works without a backend.
 * - "live" extensions hit the real `deploy.extension` tRPC router.
 *
 * The hook unions both sources and tags every row with its `source`. Mutations
 * route to the right backend based on the extension's *effective* mode (the
 * manifest mode, downgraded to "preview" when the `extensionsLive` flag is off).
 */
"use client";

import { useFlag } from "@/lib/flags/provider";
import { trpc } from "@/lib/trpc/client";
import { newId } from "@unkey/id";
import { useCallback, useEffect, useMemo, useSyncExternalStore } from "react";
import {
  EXTENSIONS,
  type Extension,
  type ExtensionConfigState,
  type ExtensionMode,
} from "./registry";

export type InstallationStatus = "active" | "degraded" | "disabled" | "verifying" | "failed";

export type Installation = {
  id: string;
  extensionSlug: string;
  projectId: string;
  instanceName: string;
  status: InstallationStatus;
  config: ExtensionConfigState;
  oauthConnected: boolean;
  installedAt: string;
  lastEventAt?: string;
  /** Which backend owns this installation. */
  source: ExtensionMode;
};

const STORAGE_KEY = "unkey:extensions:installations:v2";

type Store = {
  installations: Installation[];
};

const listeners = new Set<() => void>();
let cached: Store | null = null;

function emptyStore(): Store {
  return { installations: [] };
}

function readStore(): Store {
  if (cached) {
    return cached;
  }
  if (typeof window === "undefined") {
    cached = emptyStore();
    return cached;
  }
  const raw = window.localStorage.getItem(STORAGE_KEY);
  if (!raw) {
    cached = emptyStore();
    return cached;
  }
  try {
    const parsed = JSON.parse(raw) as Store;
    cached =
      Array.isArray(parsed?.installations) === false
        ? emptyStore()
        : { installations: parsed.installations };
  } catch {
    cached = emptyStore();
  }
  return cached;
}

function writeStore(next: Store): void {
  cached = next;
  if (typeof window !== "undefined") {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
  }
  for (const listener of listeners) {
    listener();
  }
}

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

function getServerSnapshot(): Store {
  return emptyStore();
}

type CreateInput = Omit<Installation, "id" | "installedAt" | "projectId" | "status" | "source">;

/**
 * Effective install mode given the manifest mode and the `extensionsLive` flag.
 *
 * When the flag is off, live extensions degrade to preview so the dashboard
 * can ship marketplace UI everywhere without exposing real backend wiring.
 */
export function useEffectiveMode(extension: Pick<Extension, "mode">): ExtensionMode {
  const live = useFlag("extensionsLive");
  return extension.mode === "live" && !live ? "preview" : extension.mode;
}

export function useInstallations(projectId: string) {
  const live = useFlag("extensionsLive");
  const store = useSyncExternalStore(subscribe, readStore, getServerSnapshot);
  const utils = trpc.useUtils();

  // Cross-tab sync for the localStorage half.
  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }
    const onStorage = (event: StorageEvent) => {
      if (event.key === STORAGE_KEY) {
        cached = null;
        for (const listener of listeners) {
          listener();
        }
      }
    };
    window.addEventListener("storage", onStorage);
    return () => window.removeEventListener("storage", onStorage);
  }, []);

  const liveQuery = trpc.deploy.extension.list.useQuery(
    { projectId },
    { enabled: live, staleTime: 30_000 },
  );

  const installMutation = trpc.deploy.extension.install.useMutation();
  const updateMutation = trpc.deploy.extension.update.useMutation();
  const uninstallMutation = trpc.deploy.extension.uninstall.useMutation();
  const setEnabledMutation = trpc.deploy.extension.setEnabled.useMutation();

  const installations = useMemo<Installation[]>(() => {
    const previewRows = store.installations.filter((i) => i.projectId === projectId);
    const liveRows: Installation[] = (liveQuery.data ?? []).map((row) => ({
      id: row.id,
      extensionSlug: row.extensionSlug,
      projectId,
      instanceName: row.instanceName,
      status: row.status,
      config: row.config,
      oauthConnected: row.oauthConnected,
      installedAt: row.installedAt,
      lastEventAt: row.lastEventAt,
      source: "live",
    }));
    return [...liveRows, ...previewRows];
  }, [store.installations, liveQuery.data, projectId]);

  const create = useCallback(
    async (input: CreateInput): Promise<Installation> => {
      const mode = effectiveMode(input.extensionSlug, live);
      if (mode === "live") {
        const { id } = await installMutation.mutateAsync({
          projectId,
          extensionSlug: input.extensionSlug,
          instanceName: input.instanceName,
          config: input.config,
        });
        await utils.deploy.extension.list.invalidate({ projectId });
        return {
          ...input,
          id,
          projectId,
          installedAt: new Date().toISOString(),
          status: "active",
          source: "live",
        };
      }
      const installation: Installation = {
        ...input,
        id: newId("extensionInstallation"),
        projectId,
        installedAt: new Date().toISOString(),
        status: "active",
        source: "preview",
      };
      writeStore({ installations: [...readStore().installations, installation] });
      return installation;
    },
    [projectId, live, installMutation, utils],
  );

  const remove = useCallback(
    async (id: string) => {
      const target = installations.find((i) => i.id === id);
      if (target?.source === "live") {
        await uninstallMutation.mutateAsync({ id });
        await utils.deploy.extension.list.invalidate({ projectId });
        return;
      }
      writeStore({
        installations: readStore().installations.filter((i) => i.id !== id),
      });
    },
    [installations, projectId, uninstallMutation, utils],
  );

  const update = useCallback(
    async (id: string, patch: Partial<Installation>) => {
      const target = installations.find((i) => i.id === id);
      if (target?.source === "live") {
        await updateMutation.mutateAsync({
          id,
          ...(patch.instanceName !== undefined ? { instanceName: patch.instanceName } : {}),
          ...(patch.config !== undefined ? { config: patch.config } : {}),
        });
        await utils.deploy.extension.list.invalidate({ projectId });
        return;
      }
      writeStore({
        installations: readStore().installations.map((i) => (i.id === id ? { ...i, ...patch } : i)),
      });
    },
    [installations, projectId, updateMutation, utils],
  );

  const setEnabled = useCallback(
    async (id: string, enabled: boolean) => {
      const target = installations.find((i) => i.id === id);
      if (!target) {
        return;
      }
      const nextStatus: InstallationStatus = enabled ? "active" : "disabled";
      if (target.source === "live") {
        await setEnabledMutation.mutateAsync({ id, enabled });
        await utils.deploy.extension.list.invalidate({ projectId });
        return;
      }
      writeStore({
        installations: readStore().installations.map((i) =>
          i.id === id ? { ...i, status: nextStatus } : i,
        ),
      });
    },
    [installations, projectId, setEnabledMutation, utils],
  );

  return { installations, create, remove, update, setEnabled };
}

function effectiveMode(slug: string, liveFlag: boolean): ExtensionMode {
  const manifestMode = EXTENSIONS.find((e) => e.slug === slug)?.mode ?? "preview";
  return manifestMode === "live" && !liveFlag ? "preview" : manifestMode;
}

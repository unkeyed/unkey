/**
 * Live-extension provider registry.
 *
 * Live extensions (the ones backed by real infrastructure, e.g. log drains)
 * register a provider here. The `deploy.extension` tRPC router invokes the
 * provider hooks after the corresponding DB op so each extension owns its
 * own side effects without the router needing to know which slugs are special.
 *
 * Adding a new live extension = drop one entry into PROVIDERS pointing at a
 * provider module under ./providers/<slug>.ts. No conditionals anywhere else.
 */
import type { schema } from "@unkey/db";
import { logDrainProvider } from "./providers/log-drain";

type Installation = typeof schema.extensionInstallations.$inferSelect;

export type ProviderContext = {
  workspaceId: string;
};

/**
 * Result of a pre-install (or pre-update) sanity check. `error` is shown to the
 * user verbatim so providers should keep messages short and actionable.
 */
export type VerifyResult = { ok: true } | { ok: false; status?: number; error: string };

export type LiveProvider = {
  /** Called after a successful insert into `extension_installations`. */
  onInstall(ctx: ProviderContext, installation: Installation): Promise<void>;
  /** Called after a successful soft-delete of the installation row. */
  onUninstall(ctx: ProviderContext, installation: Installation): Promise<void>;
  /** Called after a successful update. Optional — many providers don't need it. */
  onUpdate?(
    ctx: ProviderContext,
    installation: Installation,
    patch: { instanceName?: string; config?: Installation["config"] },
  ): Promise<void>;
  /**
   * Called when status flips between `active` and `disabled`. Providers use
   * this to pause/resume runtime work without tearing down the row.
   */
  onSetEnabled?(ctx: ProviderContext, installation: Installation, enabled: boolean): Promise<void>;
  /**
   * Pre-install (or pre-update) connectivity check. The wizard calls this
   * before persisting so customers see auth/dataset/endpoint errors at setup
   * time instead of staring at a paused integration later. Receives the raw
   * config form state so it can run without an installation row existing yet.
   */
  verify?(ctx: ProviderContext, config: Installation["config"]): Promise<VerifyResult>;
};

/**
 * Slug → provider. Populated as live extensions land.
 *
 * The "axiom" extension provisions a real `log_drains` row keyed by the
 * installation id; the provider hooks below own that lifecycle so the
 * extension router stays generic.
 */
const PROVIDERS: Partial<Record<string, LiveProvider>> = {
  axiom: logDrainProvider,
};

export function getProvider(slug: string): LiveProvider | undefined {
  return PROVIDERS[slug];
}

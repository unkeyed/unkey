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

type Installation = typeof schema.extensionInstallations.$inferSelect;

export type ProviderContext = {
  workspaceId: string;
};

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
};

/**
 * Slug → provider. Populated as live extensions land.
 *
 * `axiom-logdrain` will register its provider here once the log_drains stack
 * merges; the provider opens a `log_drains` row keyed by the installation id.
 */
const PROVIDERS: Partial<Record<string, LiveProvider>> = {};

export function getProvider(slug: string): LiveProvider | undefined {
  return PROVIDERS[slug];
}

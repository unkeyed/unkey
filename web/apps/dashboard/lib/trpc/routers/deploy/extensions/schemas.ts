import { z } from "zod";

/**
 * Raw config-form state. Mirrors lib/extensions/registry#ExtensionConfigState.
 *
 * Validation against the per-extension manifest happens client-side in the
 * install wizard; the backend persists whatever shape the manifest declares.
 * That means new fields/types ship without a backend deploy, at the cost of
 * the DB not enforcing structure.
 */
export const extensionConfigSchema = z.record(
  z.string(),
  z.union([z.string(), z.boolean(), z.array(z.string())]),
);

export const installationStatusSchema = z.enum([
  "active",
  "degraded",
  "disabled",
  "verifying",
  "failed",
]);

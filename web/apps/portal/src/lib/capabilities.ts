import { z } from "zod";

/** Capabilities accepted in persisted portal session grants. */
export const PORTAL_CAPABILITIES = [
  "keys:read",
  "keys:create",
  "keys:reroll",
  "analytics:read",
] as const;

const portalCapabilitySchema = z.enum(PORTAL_CAPABILITIES);

/** One product action granted to a portal session. */
export type PortalCapability = z.infer<typeof portalCapabilitySchema>;

/** Validates the complete authorization grant stored on a portal session. */
export const portalSessionGrantSchema = z.object({
  keyspaceIds: z.array(z.string().min(1)),
  permissions: z.array(portalCapabilitySchema).min(1),
});

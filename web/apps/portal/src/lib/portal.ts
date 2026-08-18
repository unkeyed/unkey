import "@tanstack/react-start/server-only";
import { eq } from "@unkey/db";
import { db, schema } from "./db";

export type PortalBranding = {
  logoUrl: string | null;
  primaryColor: string | null;
};

export type Portal = {
  id: string;
  slug: string;
  enabled: boolean;
  branding: PortalBranding | null;
};

/**
 * Load a portal and its branding by id. Called server-side on page load — the
 * portal id comes from the session.
 *
 * Branding lives on the portal row, so this is one read rather than a join. The
 * columns are typed and length bounded by the database, so no parsing step is
 * needed; a portal with neither field set is reported as "no branding" so the
 * caller does not have to distinguish empty from absent.
 */
export async function loadPortal(portalId: string): Promise<Portal | null> {
  const config = await db.query.portals.findFirst({
    where: eq(schema.portals.id, portalId),
    columns: {
      id: true,
      slug: true,
      enabled: true,
      logoUrl: true,
      primaryColor: true,
    },
  });

  if (!config) {
    return null;
  }

  const hasBranding = config.logoUrl !== null || config.primaryColor !== null;

  return {
    id: config.id,
    slug: config.slug,
    enabled: config.enabled,
    branding: hasBranding ? { logoUrl: config.logoUrl, primaryColor: config.primaryColor } : null,
  };
}

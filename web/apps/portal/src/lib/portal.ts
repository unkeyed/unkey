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
  returnUrl: string | null;
  branding: PortalBranding;
};

/**
 * Load a portal from the database by portal_id.
 * Called server-side on page load; the portal_id comes from the session.
 */
export async function loadPortal(portalId: string): Promise<Portal | null> {
  const portal = await db.query.portals.findFirst({
    where: eq(schema.portals.id, portalId),
    columns: {
      id: true,
      slug: true,
      enabled: true,
      returnUrl: true,
      logoUrl: true,
      primaryColor: true,
    },
  });

  if (!portal) {
    return null;
  }

  return {
    id: portal.id,
    slug: portal.slug,
    enabled: portal.enabled,
    returnUrl: portal.returnUrl,
    branding: {
      logoUrl: portal.logoUrl,
      primaryColor: portal.primaryColor,
    },
  };
}

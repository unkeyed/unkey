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
  branding: PortalBranding | null;
};

/**
 * Branding is a JSON column, so it is never validated by the database. Parse it
 * at this boundary rather than trusting the stored shape: an unexpected value
 * degrades to "no branding" instead of reaching the UI as a bad CSS variable.
 */
function readBranding(raw: unknown): PortalBranding | null {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
    return null;
  }

  const { logoUrl, primaryColor } = raw as { logoUrl?: unknown; primaryColor?: unknown };
  const branding: PortalBranding = {
    logoUrl: typeof logoUrl === "string" && logoUrl !== "" ? logoUrl : null,
    primaryColor: typeof primaryColor === "string" && primaryColor !== "" ? primaryColor : null,
  };

  // An object carrying neither field is indistinguishable from no branding.
  if (branding.logoUrl === null && branding.primaryColor === null) {
    return null;
  }

  return branding;
}

/**
 * Load a portal and its branding by id. Called server-side on page load — the
 * portal id comes from the session.
 *
 * Branding lives on the portal row, so this is one read rather than a join.
 */
export async function loadPortal(portalId: string): Promise<Portal | null> {
  const config = await db.query.portals.findFirst({
    where: eq(schema.portals.id, portalId),
    columns: {
      id: true,
      slug: true,
      enabled: true,
      returnUrl: true,
      branding: true,
    },
  });

  if (!config) {
    return null;
  }

  return {
    id: config.id,
    slug: config.slug,
    enabled: config.enabled,
    returnUrl: config.returnUrl,
    branding: readBranding(config.branding),
  };
}

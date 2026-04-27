import "@tanstack/react-start/server-only";
import { eq } from "@unkey/db";
import { db, schema } from "./db";

export type PortalBranding = {
  logoUrl: string | null;
  primaryColor: string | null;
  secondaryColor: string | null;
};

export type PortalConfig = {
  id: string;
  enabled: boolean;
  returnUrl: string | null;
  branding: PortalBranding | null;
};

/**
 * Load portal configuration and branding from the database by portal_config_id.
 * Called server-side on page load — the portal_config_id comes from the session.
 */
export async function loadPortalConfig(portalConfigId: string): Promise<PortalConfig | null> {
  const config = await db.query.portalConfigurations.findFirst({
    where: eq(schema.portalConfigurations.id, portalConfigId),
    columns: {
      id: true,
      enabled: true,
      returnUrl: true,
    },
    with: {
      branding: {
        columns: {
          logoUrl: true,
          primaryColor: true,
          secondaryColor: true,
        },
      },
    },
  });

  if (!config) {
    return null;
  }

  return {
    id: config.id,
    enabled: config.enabled,
    returnUrl: config.returnUrl,
    branding: config.branding ?? null,
  };
}

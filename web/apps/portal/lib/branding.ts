import { db } from "@/lib/db";
import { eq, schema } from "@unkey/db";

const DEFAULT_PRIMARY_COLOR = "#6366f1";
const DEFAULT_SECONDARY_COLOR = "#f8fafc";

type PortalBranding = {
  logoUrl: string | null;
  primaryColor: string;
  secondaryColor: string;
};

/**
 * Load branding for a portal configuration from the database.
 * Falls back to Unkey defaults when no branding is configured.
 */
export async function loadBranding(portalConfigId: string): Promise<PortalBranding> {
  const branding = await db.query.portalBranding.findFirst({
    where: eq(schema.portalBranding.portalConfigId, portalConfigId),
  });

  return {
    logoUrl: branding?.logoUrl ?? null,
    primaryColor: branding?.primaryColor ?? DEFAULT_PRIMARY_COLOR,
    secondaryColor: branding?.secondaryColor ?? DEFAULT_SECONDARY_COLOR,
  };
}

/**
 * Generate a CSS custom properties style object from branding config.
 * Applied to the <html> element to make colors available via Tailwind.
 */
export function brandingToCssVars(branding: PortalBranding): Record<string, string> {
  return {
    "--portal-primary": branding.primaryColor,
    "--portal-secondary": branding.secondaryColor,
  };
}

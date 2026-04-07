import { db } from "@/lib/db";
import { and, eq, schema } from "@unkey/db";

type PortalConfig = {
  id: string;
  workspaceId: string;
  enabled: boolean;
  returnUrl: string | null;
};

/**
 * Resolve the portal configuration for the current hostname.
 * Looks up the frontline route by FQDN, then loads the portal config.
 */
export async function resolvePortalConfig(hostname: string): Promise<PortalConfig | null> {
  const route = await db.query.frontlineRoutes.findFirst({
    where: and(
      eq(schema.frontlineRoutes.fullyQualifiedDomainName, hostname),
      eq(schema.frontlineRoutes.routeType, "portal"),
    ),
  });

  if (!route?.portalConfigId) {
    return null;
  }

  const config = await db.query.portalConfigurations.findFirst({
    where: eq(schema.portalConfigurations.id, route.portalConfigId),
  });

  if (!config) {
    return null;
  }

  return {
    id: config.id,
    workspaceId: config.workspaceId,
    enabled: config.enabled,
    returnUrl: config.returnUrl,
  };
}

import { Outlet, createFileRoute, redirect } from "@tanstack/react-router";
import { PortalHeader } from "~/components/portal-header";
import { PreviewBanner } from "~/components/preview-banner";
import { loadPortalConfig } from "~/lib/portal-config";
import { getSession } from "~/lib/session";

export const Route = createFileRoute("/_portal")({
  beforeLoad: async () => {
    const session = await getSession();
    if (!session) {
      throw redirect({ to: "/" });
    }

    let config = null;
    try {
      config = await loadPortalConfig(session.portalConfigId);
    } catch (err) {
      console.error("Failed to load portal config", { portalConfigId: session.portalConfigId, err });
    }

    return { session, portalConfig: config };
  },
  component: PortalLayout,
});

function PortalLayout() {
  const { session, portalConfig } = Route.useRouteContext();

  // Inject branding CSS variables so components can use them.
  const brandingStyle: Record<string, string> = {};
  if (portalConfig?.branding?.primaryColor) {
    brandingStyle["--portal-primary"] = portalConfig.branding.primaryColor;
  }
  if (portalConfig?.branding?.secondaryColor) {
    brandingStyle["--portal-secondary"] = portalConfig.branding.secondaryColor;
  }

  return (
    <div style={brandingStyle}>
      {session.preview && <PreviewBanner />}
      <PortalHeader
        permissions={session.permissions}
        logoUrl={portalConfig?.branding?.logoUrl ?? undefined}
      />
      <Outlet />
    </div>
  );
}

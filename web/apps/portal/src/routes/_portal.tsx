import { Outlet, createFileRoute, redirect } from "@tanstack/react-router";
import { PortalHeader } from "~/components/portal-header";
import { PreviewBanner } from "~/components/preview-banner";
import { getSessionWithConfig } from "~/lib/session";

export const Route = createFileRoute("/_portal")({
  beforeLoad: async () => {
    const result = await getSessionWithConfig();
    if (!result) {
      throw redirect({ to: "/" });
    }
    return { session: result.session, portalConfig: result.config };
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

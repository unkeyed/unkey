import { Outlet, createFileRoute, redirect } from "@tanstack/react-router";
import { PortalFooter } from "~/components/portal-footer";
import { PortalHeader } from "~/components/portal-header";
import { PreviewBanner } from "~/components/preview-banner";
import { onPrimaryColor } from "~/lib/branding";
import { getSessionWithConfig } from "~/lib/session";

export const Route = createFileRoute("/_portal")({
  beforeLoad: async () => {
    const result = await getSessionWithConfig();
    if (!result) {
      throw redirect({ to: "/" });
    }
    return {
      session: result.session,
      portalConfig: result.config,
      logsRetentionDays: result.logsRetentionDays,
    };
  },
  component: PortalLayout,
});

function PortalLayout() {
  const { session, portalConfig } = Route.useRouteContext();

  // Inject branding CSS variables so components can use them. The foreground is
  // always set (even without a brand color) so surfaces painted with
  // --portal-primary — which falls back to the dark gray-12 — get readable text.
  const brandingStyle: Record<string, string> = {
    "--portal-primary-foreground": onPrimaryColor(portalConfig?.branding?.primaryColor),
  };
  if (portalConfig?.branding?.primaryColor) {
    brandingStyle["--portal-primary"] = portalConfig.branding.primaryColor;
  }

  return (
    <div style={brandingStyle} className="flex min-h-screen flex-col bg-background">
      {session.preview && <PreviewBanner />}
      <PortalHeader
        permissions={session.permissions}
        logoUrl={portalConfig?.branding?.logoUrl ?? undefined}
        returnUrl={portalConfig?.returnUrl ?? undefined}
        appName={portalConfig?.slug ?? undefined}
      />
      <div className="flex-1">
        <Outlet />
      </div>
      <PortalFooter />
    </div>
  );
}

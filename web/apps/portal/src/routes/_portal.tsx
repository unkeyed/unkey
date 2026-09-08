import { Outlet, createFileRoute, redirect } from "@tanstack/react-router";
// Deep import on purpose: the `@unkey/ui` barrel side-effect-imports the full
// stylesheet, which this app does not want in its bundle for one pure helper.
import { onPrimaryColor } from "@unkey/ui/src/lib/branding";
import { PortalFooter } from "~/components/portal-footer";
import { PortalHeader } from "~/components/portal-header";
import { PreviewBanner } from "~/components/preview-banner";
import { getSessionWithConfig } from "~/lib/session";

export const Route = createFileRoute("/_portal")({
  beforeLoad: async () => {
    const result = await getSessionWithConfig();
    if (!result) {
      throw redirect({ to: "/" });
    }
    return {
      session: result.session,
      portal: result.config,
      logsRetentionDays: result.logsRetentionDays,
    };
  },
  component: PortalLayout,
});

function PortalLayout() {
  const { session, portal } = Route.useRouteContext();

  // Inject branding CSS variables so components can use them. The foreground is
  // always set (even without a brand color) so surfaces painted with
  // --portal-primary — which falls back to the dark gray-12 — get readable text.
  const brandingStyle: Record<string, string> = {
    "--portal-primary-foreground": onPrimaryColor(portal?.branding?.primaryColor),
  };
  if (portal?.branding?.primaryColor) {
    brandingStyle["--portal-primary"] = portal.branding.primaryColor;
  }

  return (
    <div style={brandingStyle} className="flex min-h-screen flex-col bg-background">
      {session.preview && <PreviewBanner />}
      <PortalHeader
        logoUrl={portal?.branding?.logoUrl ?? undefined}
        returnUrl={session.returnUrl ?? undefined}
        appName={portal?.displayName ?? undefined}
      />
      <div className="flex-1">
        <Outlet />
      </div>
      <PortalFooter />
    </div>
  );
}

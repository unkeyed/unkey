// Deep import on purpose: the `@unkey/ui` barrel side-effect-imports the full
// stylesheet, which this app does not want in its bundle for one pure helper.
import { onPrimaryColor } from "@unkey/ui/src/lib/branding";
import type { ReactNode } from "react";
import { PortalFooter } from "~/components/portal-footer";
import { PortalHeader } from "~/components/portal-header";
import type { Portal } from "~/lib/portal";
import type { SessionData } from "~/lib/session";

type PortalShellProps = {
  session: SessionData | null;
  portal: Portal | null;
  pending?: boolean;
  banner?: ReactNode;
  children: ReactNode;
};

export function PortalShell({ session, portal, pending, banner, children }: PortalShellProps) {
  // The foreground is always set (even without a brand color) so surfaces
  // painted with --portal-primary — which falls back to the dark gray-12 — get
  // readable text.
  const brandingStyle: Record<string, string> = {
    "--portal-primary-foreground": onPrimaryColor(portal?.branding?.primaryColor),
  };
  if (portal?.branding?.primaryColor) {
    brandingStyle["--portal-primary"] = portal.branding.primaryColor;
  }

  return (
    <div style={brandingStyle} className="portal-shell flex min-h-screen flex-col bg-background">
      {banner}
      {session ? (
        <PortalHeader
          logoUrl={portal?.branding?.logoUrl ?? undefined}
          returnUrl={session.returnUrl ?? undefined}
          appName={portal?.displayName ?? undefined}
        />
      ) : (
        pending && <div className="min-h-14 bg-[var(--portal-header-bg)]" />
      )}
      {children}
      {session ? <PortalFooter /> : pending && <div className="min-h-14 bg-gray-3/50" />}
    </div>
  );
}

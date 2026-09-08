import { Outlet, createFileRoute, redirect } from "@tanstack/react-router";
import { PortalShell } from "~/components/portal-shell";
import { PreviewBanner } from "~/components/preview-banner";
import { sessionQueryOptions } from "~/lib/session";

export const Route = createFileRoute("/_portal")({
  beforeLoad: async ({ context }) => {
    const result = await context.queryClient.ensureQueryData(sessionQueryOptions);
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

  return (
    <PortalShell
      session={session}
      portal={portal}
      banner={session.preview ? <PreviewBanner /> : null}
    >
      <div className="flex-1">
        <Outlet />
      </div>
    </PortalShell>
  );
}

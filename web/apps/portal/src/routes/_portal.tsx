import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";
import { getSessionToken } from "~/lib/session";
import { PortalHeader } from "~/components/portal-header";

export const Route = createFileRoute("/_portal")({
  beforeLoad: async () => {
    const token = await getSessionToken();
    if (!token) {
      throw redirect({ to: "/" });
    }
    return { sessionToken: token };
  },
  component: PortalLayout,
});

function PortalLayout() {
  return (
    <>
      <PortalHeader />
      <Outlet />
    </>
  );
}

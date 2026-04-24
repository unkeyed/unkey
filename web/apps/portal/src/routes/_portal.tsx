import { Outlet, createFileRoute, redirect } from "@tanstack/react-router";
import { getSessionToken } from "~/lib/session";

export const Route = createFileRoute("/_portal")({
  beforeLoad: async () => {
    const token = await getSessionToken();
    if (!token) {
      throw redirect({ to: "/" });
    }
    return { sessionToken: token };
  },
  component: Outlet,
});

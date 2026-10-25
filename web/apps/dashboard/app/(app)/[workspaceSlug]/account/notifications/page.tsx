import { env, workosAuthEnv } from "@/lib/env";
import { routes } from "@/lib/navigation/routes";
import { redirect } from "next/navigation";
import { AccountShell } from "../account-shell";
import { AccountUnavailable } from "../account-unavailable";
import { PersonalNotifications } from "./personal-notifications";

export default async function PersonalNotificationsPage() {
  if (env().AUTH_PROVIDER !== "local") {
    workosAuthEnv();
    const { withAuth } = await import("@workos-inc/authkit-nextjs");
    const session = await withAuth({ ensureSignedIn: true });
    if (!session.user) {
      redirect(routes.auth.signIn());
    }

    if (session.impersonator) {
      return (
        <AccountShell>
          <AccountUnavailable reason="impersonation" />
        </AccountShell>
      );
    }
  }

  return <PersonalNotifications />;
}

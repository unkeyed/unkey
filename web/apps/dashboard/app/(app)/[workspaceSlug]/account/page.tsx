import { env, workosAuthEnv } from "@/lib/env";
import { routes } from "@/lib/navigation/routes";
import { redirect } from "next/navigation";
import { AccountShell } from "./account-shell";
import { AccountUnavailable } from "./account-unavailable";

export default async function AccountPage() {
  if (env().AUTH_PROVIDER === "local") {
    return (
      <AccountShell>
        <AccountUnavailable reason="local" />
      </AccountShell>
    );
  }

  workosAuthEnv();
  const [{ withAuth }, { ManagedAccount }] = await Promise.all([
    import("@workos-inc/authkit-nextjs"),
    import("./managed-account"),
  ]);
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

  return <ManagedAccount />;
}

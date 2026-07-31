import { getWorkOSSession } from "@/lib/auth/workos-session";
import { env } from "@/lib/env";
import type React from "react";

export async function AuthProvider({ children }: { children: React.ReactNode }) {
  if (env().AUTH_PROVIDER === "local") {
    return children;
  }

  const [{ AuthKitProvider, Impersonation }, session] = await Promise.all([
    import("@workos-inc/authkit-nextjs/components"),
    getWorkOSSession(),
  ]);
  const { accessToken: _accessToken, ...initialAuth } = session;

  return (
    <AuthKitProvider initialAuth={initialAuth}>
      {children}
      <Impersonation side="top" returnTo="/auth/sign-in" />
    </AuthKitProvider>
  );
}

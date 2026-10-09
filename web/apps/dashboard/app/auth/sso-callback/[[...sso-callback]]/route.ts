import { expireLegacySession } from "@/lib/auth/legacy-session";
import { logManagedAuthOutcome } from "@/lib/auth/telemetry";
import { env, workosAuthEnv } from "@/lib/env";
import { type NextRequest, NextResponse } from "next/server";

async function handleLocalCallback(request: NextRequest, baseURL?: string): Promise<Response> {
  return NextResponse.redirect(new URL("/apis", baseURL ?? request.url));
}

export async function GET(request: NextRequest): Promise<Response> {
  const { AUTH_PROVIDER: authProvider, DASHBOARD_BASE_URL: baseURL } = env();
  if (authProvider === "local") {
    return handleLocalCallback(request, baseURL);
  }

  workosAuthEnv();
  const { handleAuth } = await import("@workos-inc/authkit-nextjs");
  const response = await handleAuth({
    baseURL,
    returnPathname: "/apis",
    onSuccess: () => {
      logManagedAuthOutcome("callback", "success");
    },
    onError: ({ request: failedRequest }) => {
      logManagedAuthOutcome("callback", "failure");
      return NextResponse.redirect(
        new URL("/auth/error?reason=callback", baseURL ?? failedRequest.url),
      );
    },
  })(request);

  return expireLegacySession(request, response);
}

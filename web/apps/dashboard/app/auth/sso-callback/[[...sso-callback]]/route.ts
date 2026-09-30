import { expireLegacySession } from "@/lib/auth/legacy-session";
import { logManagedAuthOutcome } from "@/lib/auth/telemetry";
import { env, workosAuthEnv } from "@/lib/env";
import { type NextRequest, NextResponse } from "next/server";

async function handleLocalCallback(request: NextRequest): Promise<Response> {
  return NextResponse.redirect(new URL("/apis", request.url));
}

export async function GET(request: NextRequest): Promise<Response> {
  if (env().AUTH_PROVIDER === "local") {
    return handleLocalCallback(request);
  }

  workosAuthEnv();
  const { handleAuth } = await import("@workos-inc/authkit-nextjs");
  const response = await handleAuth({
    returnPathname: "/apis",
    onSuccess: () => {
      logManagedAuthOutcome("callback", "success");
    },
    onError: ({ request: failedRequest }) => {
      logManagedAuthOutcome("callback", "failure");
      return NextResponse.redirect(new URL("/auth/error?reason=callback", failedRequest.url));
    },
  })(request);

  return expireLegacySession(request, response);
}

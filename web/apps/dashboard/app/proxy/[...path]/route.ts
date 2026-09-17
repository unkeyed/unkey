import { getAuth } from "@/lib/auth/get-auth";
import { auth as authProvider } from "@/lib/auth/server";
import { env } from "@/lib/env";
import { SignJWT } from "jose";
import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";

type RouteContext = {
  params: Promise<{
    path: string[];
  }>;
};

export async function POST(req: NextRequest, ctx: RouteContext): Promise<NextResponse> {
  if (req.headers.has("authorization")) {
    return apiError(
      400,
      errorType.invalidInput,
      "Bad Request",
      "Dashboard proxy requests must not include an Authorization header.",
    );
  }

  const auth = await getAuth(req);
  if (!auth.userId || !auth.orgId) {
    return apiError(
      401,
      errorType.authenticationMissing,
      "Unauthorized",
      "Authentication required.",
    );
  }
  const orgId = auth.orgId;
  const userId = auth.userId;

  let bearerToken: string | null | undefined = auth.accessToken;
  if (!bearerToken) {
    if (!auth.role) {
      return apiError(403, errorType.forbidden, "Forbidden", "Role required.");
    }

    const user = await authProvider.getUser(userId);
    const actorName = user?.fullName ?? user?.email ?? userId;
    bearerToken = await mintProxyJWT({
      orgId,
      role: auth.role,
      subject: userId,
      name: actorName,
    }).catch((error) => {
      console.error("Failed to mint dashboard proxy JWT", { error });
      return null;
    });
  }
  if (!bearerToken) {
    return apiError(
      500,
      errorType.unexpectedError,
      "Internal Server Error",
      "Dashboard proxy is not configured.",
    );
  }

  const { path } = await ctx.params;

  // A leading "//" or backslash in a segment is parsed as an authority
  // delimiter and rewrites the upstream origin, so reject segments carrying a
  // slash, backslash, or control character.
  for (const segment of path) {
    const hasControlChar = [...segment].some((char) => {
      const code = char.charCodeAt(0);
      return code <= 0x1f || code === 0x7f;
    });
    if (/[\\/]/.test(segment) || hasControlChar) {
      return apiError(400, errorType.invalidInput, "Bad Request", "Invalid proxy path.");
    }
  }

  const baseURL = new URL(env().UNKEY_API_URL);
  const upstreamURL = new URL(`/${path.join("/")}`, baseURL);
  upstreamURL.search = req.nextUrl.search;

  // Defense in depth: never let the constructed URL leave the upstream origin.
  if (upstreamURL.origin !== baseURL.origin) {
    return apiError(400, errorType.invalidInput, "Bad Request", "Invalid proxy path.");
  }

  const headers = upstreamRequestHeaders(req);
  headers.set("authorization", `Bearer ${bearerToken}`);

  const body = await req.arrayBuffer();
  const upstream = await fetch(upstreamURL, {
    method: "POST",
    headers,
    body,
    // Don't auto-follow redirects; the origin check only guards the initial
    // target, so a 3xx could still reach another host with the bearer attached.
    redirect: "manual",
    signal: AbortSignal.timeout(10_000),
  }).catch((error) => {
    console.error("Dashboard proxy request failed", { error, upstreamURL: upstreamURL.toString() });
    return null;
  });
  if (!upstream) {
    return apiError(
      502,
      errorType.serviceUnavailable,
      "Bad Gateway",
      "Upstream API request failed.",
    );
  }

  const responseHeaders = new Headers(upstream.headers);
  for (const header of hopByHopResponseHeaders) {
    responseHeaders.delete(header);
  }

  return new NextResponse(upstream.body, {
    status: upstream.status,
    statusText: upstream.statusText,
    headers: responseHeaders,
  });
}

const errorType = {
  invalidInput: "https://unkey.com/docs/errors/unkey/application/invalid_input",
  authenticationMissing: "https://unkey.com/docs/errors/unkey/authentication/missing",
  forbidden: "https://unkey.com/docs/errors/unkey/authorization/forbidden",
  unexpectedError: "https://unkey.com/docs/errors/unkey/application/unexpected_error",
  serviceUnavailable: "https://unkey.com/docs/errors/unkey/application/service_unavailable",
} as const;

// The dashboard's generated client parses every error response against svc/api's
// problem-details envelope. A bare `{ error: string }` body fails that parse and
// surfaces as a ResponseValidationError the UI cannot turn into a message, so
// proxy-generated errors use the same envelope.
function apiError(
  status: number,
  type: (typeof errorType)[keyof typeof errorType],
  title: string,
  detail: string,
): NextResponse {
  return NextResponse.json(
    {
      meta: { requestId: `req_${crypto.randomUUID()}` },
      error: {
        type,
        title,
        status,
        detail,
        // The 400 problem-details schema requires an errors array.
        ...(status === 400 ? { errors: [] } : {}),
      },
    },
    { status },
  );
}

function upstreamRequestHeaders(req: NextRequest): Headers {
  const headers = new Headers();
  const accept = req.headers.get("accept");
  if (accept) {
    headers.set("accept", accept);
  }
  const contentType = req.headers.get("content-type");
  if (contentType) {
    headers.set("content-type", contentType);
  }
  // Identifies the caller to the API so it can attribute deployments to the
  // dashboard rather than to a generic API client.
  headers.set("x-unkey-client", "unkey-dashboard");
  return headers;
}

const hopByHopResponseHeaders = new Set([
  "connection",
  "content-encoding",
  "content-length",
  "keep-alive",
  "proxy-authenticate",
  "te",
  "trailer",
  "transfer-encoding",
  "upgrade",
]);

// mintProxyJWT creates the short-lived credential that lets the dashboard call
// svc/api without exposing a root key to the browser. The dashboard only needs
// the active signing secret. svc/api carries the ordered verification set so old
// secrets can remain valid during a rotation window.
async function mintProxyJWT(params: {
  orgId: string;
  role: string;
  subject: string;
  name: string;
}): Promise<string> {
  const { UNKEY_JWT_SECRET: signingSecret } = env();
  if (!signingSecret) {
    throw new Error("UNKEY_JWT_SECRET must be configured for dashboard proxy signing");
  }

  const key = new TextEncoder().encode(signingSecret);
  const now = Math.floor(Date.now() / 1000);

  return (
    new SignJWT({
      org: {
        id: params.orgId,
      },
      name: params.name,
      roles: [params.role],
    })
      .setProtectedHeader({ alg: "HS256", typ: "JWT" })
      .setIssuer("app.unkey.com")
      // Mirror the WorkOS JWT template's audience so svc/api's WorkOS auth entry
      // (configured with audience = "api.unkey.com") verifies this fallback token
      // the same way it verifies a forwarded WorkOS access token.
      .setAudience(["api.unkey.com"])
      .setSubject(params.subject)
      .setIssuedAt(now)
      .setNotBefore(now)
      .setExpirationTime(now + 120)
      .sign(key)
  );
}

import { type GetAuthResult, getAuth } from "@/lib/auth/get-auth";
import { mintProxyJWT } from "@/lib/auth/mint-proxy-jwt";
import { env } from "@/lib/env";
import type { BadRequestErrorDetails, BaseError } from "@unkey/api/models/components";
import { newId } from "@unkey/id";
import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";

type RouteContext = {
  params: Promise<{
    path: string[];
  }>;
};

export async function POST(req: NextRequest, ctx: RouteContext): Promise<NextResponse> {
  if (req.headers.has("authorization")) {
    return proxyError(
      "invalidInput",
      "Dashboard proxy requests must not include an Authorization header.",
    );
  }

  const auth = await getAuth(req);
  if (!auth.userId || !auth.orgId) {
    return proxyError("unauthenticated", "Authentication required.");
  }
  const orgId = auth.orgId;
  const userId = auth.userId;
  const apiURL = env().UNKEY_API_URL;

  let bearerToken: string | null | undefined = upstreamBearerToken(auth, apiURL);
  if (!bearerToken) {
    if (!auth.role) {
      return proxyError("forbidden", "Role required.");
    }

    bearerToken = await mintProxyJWT({
      orgId,
      role: auth.role,
      subject: userId,
      name: auth.user?.fullName ?? auth.user?.email ?? userId,
    }).catch((error) => {
      console.error("Failed to mint dashboard proxy JWT", { error });
      return null;
    });
  }
  if (!bearerToken) {
    return proxyError("unexpected", "Dashboard proxy is not configured.");
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
      return proxyError("invalidInput", "Invalid proxy path.");
    }
  }

  const baseURL = new URL(apiURL);
  const upstreamURL = new URL(`/${path.join("/")}`, baseURL);
  upstreamURL.search = req.nextUrl.search;

  // Defense in depth: never let the constructed URL leave the upstream origin.
  if (upstreamURL.origin !== baseURL.origin) {
    return proxyError("invalidInput", "Invalid proxy path.");
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
    return proxyError("upstreamUnavailable", "Upstream API request failed.");
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

// The SDK only parses error bodies shaped like the API's, so proxy failures
// must use the same envelope and docs URLs as pkg/codes.
const proxyErrors = {
  invalidInput: {
    status: 400,
    title: "Bad Request",
    type: "https://unkey.com/docs/errors/unkey/application/invalid_input",
    errors: [],
  },
  unauthenticated: {
    status: 401,
    title: "Unauthorized",
    type: "https://unkey.com/docs/errors/unkey/authentication/missing",
  },
  forbidden: {
    status: 403,
    title: "Forbidden",
    type: "https://unkey.com/docs/errors/unkey/authorization/insufficient_permissions",
  },
  unexpected: {
    status: 500,
    title: "Internal Server Error",
    type: "https://unkey.com/docs/errors/unkey/application/unexpected_error",
  },
  upstreamUnavailable: {
    status: 502,
    title: "Bad Gateway",
    type: "https://unkey.com/docs/errors/unkey/application/service_unavailable",
  },
} satisfies Record<string, Omit<BaseError, "detail"> | Omit<BadRequestErrorDetails, "detail">>;

function proxyError(kind: keyof typeof proxyErrors, detail: string): NextResponse {
  const error = { ...proxyErrors[kind], detail };
  return NextResponse.json(
    { meta: { requestId: newId("request") }, error },
    { status: error.status },
  );
}

function upstreamBearerToken(auth: GetAuthResult, apiURL: string): string | undefined {
  // Local APIs verify the dashboard HS256 secret, not WorkOS JWKS. Returning
  // no token makes the caller mint that JWT for localhost, 127.0.0.1, ::1,
  // and *.unkey.local. A WorkOS access token fails those calls, including CLI
  // approval. When WorkOS is not configured, accessToken is already empty and
  // the caller mints the same JWT.
  if (env().UNKEY_JWT_SECRET && apiVerifiesDashboardJWT(apiURL)) {
    return undefined;
  }
  return auth.accessToken;
}

function apiVerifiesDashboardJWT(apiURL: string): boolean {
  let hostname: string;
  try {
    hostname = new URL(apiURL).hostname;
  } catch {
    return false;
  }
  return (
    hostname === "localhost" ||
    hostname === "127.0.0.1" ||
    hostname === "[::1]" ||
    hostname.endsWith(".unkey.local")
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

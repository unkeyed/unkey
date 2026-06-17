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
    return NextResponse.json(
      { error: "Dashboard proxy requests must not include an Authorization header." },
      { status: 400 },
    );
  }

  const auth = await getAuth(req);
  if (!auth.userId || !auth.orgId) {
    return NextResponse.json({ error: "Authentication required." }, { status: 401 });
  }
  const orgId = auth.orgId;
  const userId = auth.userId;

  let bearerToken: string | null | undefined = auth.accessToken;
  if (!bearerToken) {
    const user = await authProvider.getUser(userId);
    const actorName = user?.fullName ?? user?.email ?? userId;
    bearerToken = await mintProxyJWT({
      orgId,
      permissions: auth.permissions ?? [],
      subject: userId,
      name: actorName,
    }).catch((error) => {
      console.error("Failed to mint dashboard proxy JWT", { error });
      return null;
    });
  }
  if (!bearerToken) {
    return NextResponse.json({ error: "Dashboard proxy is not configured." }, { status: 500 });
  }

  const { path } = await ctx.params;
  const upstreamURL = new URL(`/${path.join("/")}`, env().UNKEY_API_URL);
  upstreamURL.search = req.nextUrl.search;

  const headers = upstreamRequestHeaders(req);
  headers.set("authorization", `Bearer ${bearerToken}`);

  const body = await req.arrayBuffer();
  const upstream = await fetch(upstreamURL, {
    method: "POST",
    headers,
    body,
    signal: AbortSignal.timeout(10_000),
  }).catch((error) => {
    console.error("Dashboard proxy request failed", { error, upstreamURL: upstreamURL.toString() });
    return null;
  });
  if (!upstream) {
    return NextResponse.json({ error: "Upstream API request failed." }, { status: 502 });
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
  permissions: readonly string[];
  subject: string;
  name: string;
}): Promise<string> {
  const { UNKEY_JWT_SECRET: signingSecret } = env();
  if (!signingSecret) {
    throw new Error("UNKEY_JWT_SECRET must be configured for dashboard proxy signing");
  }
  if (params.permissions.length === 0) {
    throw new Error("permissions are required for dashboard proxy signing");
  }

  const key = new TextEncoder().encode(signingSecret);
  const now = Math.floor(Date.now() / 1000);

  return new SignJWT({
    org: {
      id: params.orgId,
    },
    name: params.name,
    perms: params.permissions,
  })
    .setProtectedHeader({ alg: "HS256", typ: "JWT" })
    .setIssuer("app.unkey.com")
    // Mirror the WorkOS JWT template's audience so svc/api's WorkOS auth entry
    // (configured with audience = "api.unkey.com") verifies this fallback token
    // the same way it verifies a forwarded WorkOS access token.
    .setAudience(["app.unkey.com", "api.unkey.com"])
    .setSubject(params.subject)
    .setIssuedAt(now)
    .setNotBefore(now)
    .setExpirationTime(now + 120)
    .sign(key);
}

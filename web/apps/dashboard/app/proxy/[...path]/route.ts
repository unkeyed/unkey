import { getAuth } from "@/lib/auth/get-auth";
import { mintProxyJWT } from "@/lib/auth/proxy-jwt";
import { env } from "@/lib/env";
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
    if (!auth.role) {
      return NextResponse.json({ error: "Role required." }, { status: 403 });
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
    return NextResponse.json({ error: "Dashboard proxy is not configured." }, { status: 500 });
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
      return NextResponse.json({ error: "Invalid proxy path." }, { status: 400 });
    }
  }

  const baseURL = new URL(env().UNKEY_API_URL);
  const upstreamURL = new URL(`/${path.join("/")}`, baseURL);
  upstreamURL.search = req.nextUrl.search;

  // Defense in depth: never let the constructed URL leave the upstream origin.
  if (upstreamURL.origin !== baseURL.origin) {
    return NextResponse.json({ error: "Invalid proxy path." }, { status: 400 });
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

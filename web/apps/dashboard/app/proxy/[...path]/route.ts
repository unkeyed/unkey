import { getAuth } from "@/lib/auth/get-auth";
import { auth as authProvider } from "@/lib/auth/server";
import { db } from "@/lib/db";
import { env } from "@/lib/env";
import { SignJWT } from "jose";
import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";

type RouteContext = {
  params: Promise<{
    path: string[];
  }>;
};

const encoder = new TextEncoder();

const dashboardProxyPermissions = [
  "api.*.read_api",
  "api.*.create_api",
  "api.*.delete_api",
  "api.*.update_api",
  "api.*.create_key",
  "api.*.update_key",
  "api.*.delete_key",
  "api.*.encrypt_key",
  "api.*.decrypt_key",
  "api.*.read_key",
  "api.*.verify_key",
  "api.*.read_analytics",
  "ratelimit.*.limit",
  "ratelimit.*.create_namespace",
  "ratelimit.*.read_namespace",
  "ratelimit.*.update_namespace",
  "ratelimit.*.delete_namespace",
  "ratelimit.*.set_override",
  "ratelimit.*.read_override",
  "ratelimit.*.delete_override",
  "ratelimit.*.list_overrides",
  "rbac.*.create_permission",
  "rbac.*.update_permission",
  "rbac.*.delete_permission",
  "rbac.*.read_permission",
  "rbac.*.create_role",
  "rbac.*.update_role",
  "rbac.*.delete_role",
  "rbac.*.read_role",
  "rbac.*.add_permission_to_key",
  "rbac.*.remove_permission_from_key",
  "rbac.*.add_role_to_key",
  "rbac.*.remove_role_from_key",
  "rbac.*.add_permission_to_role",
  "rbac.*.remove_permission_from_role",
  "identity.*.create_identity",
  "identity.*.read_identity",
  "identity.*.update_identity",
  "identity.*.delete_identity",
  "project.*.create_deployment",
  "project.*.read_deployment",
  "project.*.generate_upload_url",
] as const;

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
  const user = await authProvider.getUser(userId);
  const actorName = user?.fullName || user?.email || userId;

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
    columns: {
      id: true,
      name: true,
    },
  });
  if (!workspace) {
    return NextResponse.json({ error: "Workspace not found." }, { status: 404 });
  }

  const jwt = await mintProxyJWT({
    workspaceId: workspace.id,
    subject: userId,
    name: actorName,
  }).catch((error) => {
    console.error("Failed to mint dashboard proxy JWT", { error });
    return null;
  });
  if (!jwt) {
    return NextResponse.json({ error: "Dashboard proxy is not configured." }, { status: 500 });
  }

  const { path } = await ctx.params;
  const upstreamURL = new URL(`/${path.join("/")}`, env().UNKEY_API_URL);
  upstreamURL.search = req.nextUrl.search;

  const headers = upstreamRequestHeaders(req);
  headers.set("authorization", `Bearer ${jwt}`);

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
  workspaceId: string;
  subject: string;
  name: string;
}): Promise<string> {
  const { UNKEY_JWT_SECRET: signingSecret } = env();
  if (!signingSecret) {
    throw new Error("UNKEY_JWT_SECRET must be configured for dashboard proxy signing");
  }

  const key = encoder.encode(signingSecret);
  const now = Math.floor(Date.now() / 1000);

  return new SignJWT({
    wid: params.workspaceId,
    name: params.name,
    perms: dashboardProxyPermissions,
  })
    .setProtectedHeader({ alg: "HS256", typ: "JWT" })
    .setIssuer("app.unkey.com")
    .setAudience(["api.unkey.com"])
    .setSubject(params.subject)
    .setIssuedAt(now)
    .setNotBefore(now)
    .setExpirationTime(now + 120)
    .sign(key);
}

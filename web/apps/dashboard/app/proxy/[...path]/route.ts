// Bridges dashboard requests (authenticated by a WorkOS session cookie) to
// svc/api (which only authenticates bearer tokens). For each request we mint a
// short-lived JWT bound to the caller's workspace, forward the body, and
// return the response untouched.
import { getAuth } from "@/lib/auth/get-auth";
import { db } from "@/lib/db";
import { env } from "@/lib/env";
import { mintProxyJWT } from "@/lib/proxy/mint-jwt";
import { type NextRequest, NextResponse } from "next/server";

const ALLOWED_PERMISSIONS = ["api.*.read_key"];
const UPSTREAM_TIMEOUT_MS = 10_000;

export async function POST(req: NextRequest, ctx: { params: Promise<{ path: string[] }> }) {
  if (req.headers.get("authorization")) {
    return NextResponse.json(
      { error: "Authorization header must not be set; the proxy mints its own token." },
      { status: 400 },
    );
  }

  const { userId, orgId } = await getAuth(req);
  if (!userId || !orgId) {
    return NextResponse.json({ error: "Unauthenticated" }, { status: 401 });
  }

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
  });
  if (!workspace) {
    return NextResponse.json({ error: "Workspace not found" }, { status: 404 });
  }

  const { path } = await ctx.params;
  const targetPath = `/${path.join("/")}`;

  const token = await mintProxyJWT({
    workspaceId: workspace.id,
    subjectId: userId,
    subjectName: "dashboard",
    permissions: ALLOWED_PERMISSIONS,
  });

  const upstream = await fetch(`${env().UNKEY_API_URL}${targetPath}`, {
    method: "POST",
    headers: {
      "content-type": req.headers.get("content-type") ?? "application/json",
      authorization: `Bearer ${token}`,
    },
    body: await req.text(),
    signal: AbortSignal.timeout(UPSTREAM_TIMEOUT_MS),
  });

  return new NextResponse(upstream.body, {
    status: upstream.status,
    headers: { "content-type": upstream.headers.get("content-type") ?? "application/json" },
  });
}

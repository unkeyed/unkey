import { lucia } from "@/lib/auth";
import { db } from "@/lib/db";
import { WorkOS } from "@workos-inc/node";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
// Create a Route Handler `app/callback/route.js`
import { NextRequest } from "next/server";

const workos = new WorkOS(process.env.WORKOS_API_KEY);
const clientId = process.env.WORKOS_CLIENT_ID!;

export async function GET(req: NextRequest) {
  // The authorization code returned by AuthKit
  const code = req.nextUrl.searchParams.get("code");
  if (!code) {
    return new Response("code is missing");
  }

  const { user, organizationId } = await workos.userManagement.authenticateWithCode({
    code,
    clientId,
  });

  console.error({ user });

  const session = await lucia.createSession(user.id, {});
  const sessionCookie = lucia.createSessionCookie(session.id);
  cookies().set(sessionCookie.name, sessionCookie.value, sessionCookie.attributes);

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { eq }) => eq(table.tenantId, organizationId ?? user.id),
    columns: {
      slug: true,
    },
  });
  return redirect(`/${workspace?.slug}`);
}

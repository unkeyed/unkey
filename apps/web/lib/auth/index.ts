import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { Auth } from "./interface";
import { Lucia, lucia } from "./lucia";
import { WorkOS } from "./workos";

/**
 * Return the user id or a 404 not found page.
 *
 * The auth check should already be done at a higher level, and we're just returning 404 to make typescript happy.
 */
export async function getUserId(): Promise<string> {
  const cookie = cookies().get(lucia.sessionCookieName);
  if (!cookie) {
    console.log("no cookie found, redirecting ... ");
    return redirect("/auth/sign-in");
  }

  const { session, user } = await lucia.validateSession(cookie.value);
  if (!session || !user) {
    console.log("no session or user found, redirecting ... ");
    return redirect("/auth/sign-in");
  }
  return user.id;
}
/**
 * Return the tenant id or a 404 not found page.
 *
 * The auth check should already be done at a higher level, and we're just returning 404 to make typescript happy.
 */
export async function getTenantId(): Promise<string> {
  const cookie = cookies().get(lucia.sessionCookieName);
  if (!cookie) {
    console.log("no cookie found, redirecting ... ");
    return redirect("/auth/sign-in");
  }

  const { session, user } = await lucia.validateSession(cookie.value);
  if (!session || !user) {
    console.log("no session or user found, redirecting ... ");
    return redirect("/auth/sign-in");
  }
  return user.id;
}

const workosClientId = process.env.WORKOS_CLIENT_ID;
const workosApiKey = process.env.WORKOS_API_KEY;

export const auth: Auth =
  workosApiKey && workosClientId
    ? new WorkOS({ clientId: workosClientId, apiKey: workosApiKey })
    : new Lucia(lucia);

export { lucia };

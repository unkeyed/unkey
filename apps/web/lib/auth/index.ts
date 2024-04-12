export { github } from "./github";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import type { Auth } from "./interface";
import { Impl, lucia } from "./lucia";

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

export const auth: Auth = new Impl(lucia);
export { lucia };

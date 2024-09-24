import { redirect } from "next/navigation";
import { WorkOS } from "@workos-inc/node"
import type { ServerAuth, ServerUser } from "./interface.server";

import * as authkit from "@workos-inc/authkit-nextjs";
import { env } from "@/lib/env";

export class WorkosServerAuth implements ServerAuth {
  private client: WorkOS

  constructor() {
    this.client = new WorkOS(env().WORKOS_API_KEY, {
      clientId: env().WORKOS_CLIENT_ID
    })
  }


  async getTenantId() {
    const { user } = await authkit.getUser();
    if (!user) {
      const signInUrl = await authkit.getSignInUrl();
      return redirect(signInUrl);
    }
    return user.id;
  }

  async getUser(): Promise<ServerUser | null> {
    const { user } = await authkit.getUser();

    if (!user) {
      return null;
    }

    return {
      id: user.id,
    };
  }

  async getUserFromCookie(cookie: string): Promise<ServerUser | null> {

    const session = this.client.userManagement.loadSealedSession({
      sessionData: cookie,
      cookiePassword: env().WORKOS_COOKIE_PASSWORD!
    })

    const authenticatedSession = await session.authenticate()
    if (!authenticatedSession.authenticated) {
      console.error(authenticatedSession.reason)
      return null
    }

    return authenticatedSession.user






  }
}

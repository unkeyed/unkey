import { WorkOS } from "@workos-inc/node";
import { redirect } from "next/navigation";
import type { Organisation, ServerAuth, ServerUser } from "./interface.server";

import { env } from "@/lib/env";
import * as authkit from "@workos-inc/authkit-nextjs";

export class WorkosServerAuth implements ServerAuth {
  private client: WorkOS;

  constructor() {
    this.client = new WorkOS(env().WORKOS_API_KEY, {
      clientId: env().WORKOS_CLIENT_ID,
    });
  }

  async getTenantId() {
    const user = await this.getUser();
    if (!user) {
      const signInUrl = await authkit.getSignInUrl();
      return redirect(signInUrl);
    }
    return user.id;
  }

  async getUser(response?: Response): Promise<ServerUser | null> {
    const sess = await authkit.getSession(response);
    if (!sess) {
      return null;
    }
    return { id: sess.user.id };
  }

  async listOrganisations(): Promise<Organisation[]> {
    const user = await this.getUser();
    if (!user) {
      return [];
    }

    const memberships = await this.client.userManagement.listOrganizationMemberships({
      userId: user.id,
    });
    return Promise.all(
      memberships.data.map(async (m) => {
        const org = await this.client.organizations.getOrganization(m.organizationId);
        return {
          id: org.id,
          name: org.name,
          imageUrl: org.imageUrl,
        };
      }),
    );
  }
}

import { auth } from "@clerk/nextjs";
import { redirect } from "next/navigation";
import type { ServerAuth, ServerUser } from "./interface.server";

export class ClerkServerAuth implements ServerAuth {
  async getTenantId() {
    const { user } = auth();
    if (!user) {
      return redirect("/auth/sign-in");
    }
    return user.id;
  }

  async getUser(): Promise<ServerUser | null> {
    const { user } = auth();

    if (!user) {
      return null;
    }

    return {
      id: user.id,
    };
  }
}

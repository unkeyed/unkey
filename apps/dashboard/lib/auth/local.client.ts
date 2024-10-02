import type { ClientAuth } from "./interface.client";

export class LocalClientAuth implements ClientAuth {
  useUser() {
    return {
      isSignedIn: true as const,
      isLoaded: true as const,
      user: {
        id: "user_123",
      },
    };
  }

  useOrganizationList() {
    return {
      setActive: async (_orgId: string | null) => {},
    };
  }
}

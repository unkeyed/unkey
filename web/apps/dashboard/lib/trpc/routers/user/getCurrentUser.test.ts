import { LOCAL_ORG_ID, LOCAL_USER_ID, type User } from "@/lib/auth/types";
import { afterAll, expect, it, vi } from "vitest";
import { t } from "../../trpc";
import { getCurrentUser } from "./getCurrentUser";

vi.hoisted(() => {
  vi.stubEnv("UNKEY_WORKSPACE_ID", crypto.randomUUID());
  vi.stubEnv("UNKEY_API_ID", crypto.randomUUID());
  vi.stubEnv("UNKEY_ROOT_KEY", undefined);
  vi.stubEnv("VAULT_URL", "http://localhost");
  vi.stubEnv("VAULT_TOKEN", "test");
  vi.stubEnv("AUTH_PROVIDER", "local");
});
afterAll(() => vi.unstubAllEnvs());

it.each([undefined, { email: "support@example.com", reason: "Customer support" }])(
  "retains trusted impersonation when refreshing the authenticated user: %j",
  async (impersonator) => {
    const profile: User = {
      id: LOCAL_USER_ID,
      email: "user@example.com",
      firstName: null,
      lastName: null,
      fullName: null,
      avatarUrl: null,
    };
    const caller = t.router({ getCurrentUser }).createCaller({
      req: new Request("http://localhost/api/trpc"),
      audit: { location: "unknown", userAgent: undefined },
      user: {
        id: profile.id,
        profile,
        impersonator,
      },
      tenant: { id: LOCAL_ORG_ID, role: "viewer" },
      workspace: undefined,
    });

    await expect(caller.getCurrentUser()).resolves.toEqual({
      ...profile,
      orgId: LOCAL_ORG_ID,
      role: "viewer",
      impersonator,
    });
  },
);

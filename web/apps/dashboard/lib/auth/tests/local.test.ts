import { describe, expect, it } from "vitest";
import { LocalAuthProvider } from "../local";
import { LOCAL_ORG_ID, LOCAL_ORG_ROLE, LOCAL_USER_ID } from "../types";

describe("LocalAuthProvider", () => {
  it("returns the admin role for local sessions", async () => {
    const auth = new LocalAuthProvider();

    const session = await auth.validateSession("local_session_token");

    expect(session.isValid).toBe(true);
    expect(session.userId).toBe(LOCAL_USER_ID);
    expect(session.orgId).toBe(LOCAL_ORG_ID);
    expect(session.role).toBe(LOCAL_ORG_ROLE);
  });

  it("keeps the fixed local organization administrative behavior", async () => {
    const auth = new LocalAuthProvider();

    await expect(auth.getOrg(LOCAL_ORG_ID)).resolves.toMatchObject({ id: LOCAL_ORG_ID });
    await expect(auth.getOrg("org_unknown")).rejects.toThrow("not found");
  });
});

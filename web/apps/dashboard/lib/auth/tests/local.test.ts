import { describe, expect, it } from "vitest";
import { LocalAuthProvider } from "../local";
import { LOCAL_AUTH_PERMISSIONS, LOCAL_ORG_ID, LOCAL_USER_ID } from "../types";

describe("LocalAuthProvider", () => {
  it("returns admin permissions for local sessions", async () => {
    const auth = new LocalAuthProvider();

    const session = await auth.validateSession("local_session_token");

    expect(session.isValid).toBe(true);
    expect(session.userId).toBe(LOCAL_USER_ID);
    expect(session.orgId).toBe(LOCAL_ORG_ID);
    expect(session.permissions).toEqual(LOCAL_AUTH_PERMISSIONS);
  });

  it("keeps the fixed local organization administrative behavior", async () => {
    const auth = new LocalAuthProvider();

    await expect(auth.getOrg(LOCAL_ORG_ID)).resolves.toMatchObject({ id: LOCAL_ORG_ID });
    await expect(auth.getOrg("org_unknown")).rejects.toThrow("not found");
  });
});

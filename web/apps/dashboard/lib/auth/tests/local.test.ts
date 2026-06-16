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
});

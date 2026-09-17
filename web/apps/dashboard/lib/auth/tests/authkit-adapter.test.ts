import { describe, expect, it } from "vitest";
import { mapAuthkitSession } from "../get-auth";

describe("mapAuthkitSession", () => {
  it("preserves the dashboard authorization contract", () => {
    expect(
      mapAuthkitSession({
        sessionId: "session_123",
        user: {
          id: "user_123",
          email: "james@example.com",
          firstName: "James",
          lastName: "Perkins",
          profilePictureUrl: "https://example.com/avatar.png",
        },
        organizationId: "org_123",
        role: "admin",
        permissions: ["admin:*"],
        accessToken: "access_token",
        impersonator: {
          email: "support@example.com",
          reason: "Customer support",
        },
      }),
    ).toEqual({
      userId: "user_123",
      orgId: "org_123",
      role: "admin",
      permissions: ["admin:*"],
      accessToken: "access_token",
      impersonator: {
        email: "support@example.com",
        reason: "Customer support",
      },
      user: {
        id: "user_123",
        email: "james@example.com",
        firstName: "James",
        lastName: "Perkins",
        avatarUrl: "https://example.com/avatar.png",
        fullName: "James Perkins",
      },
    });
  });

  it("maps a signed-out AuthKit result to the existing empty shape", () => {
    expect(mapAuthkitSession({ user: null })).toEqual({
      userId: null,
      orgId: null,
      role: null,
      user: null,
    });
  });
});

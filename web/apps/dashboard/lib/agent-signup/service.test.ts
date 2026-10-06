// @vitest-environment node
import { decodeJwt } from "jose";
import { beforeEach, describe, expect, it, vi } from "vitest";

const state = vi.hoisted(() => ({
  env: {
    UNKEY_API_URL: "https://api.unkey.test",
    UNKEY_AGENT_SIGNUP_JWT_PRIVATE_KEY: undefined as string | undefined,
    UNKEY_AGENT_SIGNUP_JWT_SECRET: "local-dev-agent-signup-jwt-secret-32b",
  },
  selectCalls: 0,
}));

vi.mock("@/lib/env", () => ({
  env: () => state.env,
}));

vi.mock("@/lib/db", () => ({
  primaryDb: {
    select: () => {
      state.selectCalls += 1;
      return {
        from: () => ({
          where: () => ({
            limit: async () => [{ id: "ws_123", orgId: "org_123", slug: "acme" }],
          }),
        }),
      };
    },
  },
}));

vi.mock("@/lib/auth/server", () => ({
  auth: {
    listMemberships: vi.fn(async () => ({
      data: [
        {
          organization: { id: "org_123", name: "Acme" },
          role: "admin",
          status: "active",
        },
      ],
    })),
    createTenant: vi.fn(),
    deleteTenant: vi.fn(),
  },
}));

vi.mock("@/lib/workspace/create-workspace", () => ({
  createFreeWorkspace: vi.fn(async (input: { slug: string }) => ({
    orgId: "org_new",
    workspaceId: "ws_new",
    slug: input.slug,
  })),
}));

import { auth } from "@/lib/auth/server";
import { createFreeWorkspace } from "@/lib/workspace/create-workspace";
import { AGENT_SIGNUP_AUDIENCE, AGENT_SIGNUP_ISSUER, AGENT_SIGNUP_ROLE } from "./credential";
import { createAgentWorkspace, issueAgentRootKey, liveWorkspaceWhere } from "./service";

describe("liveWorkspaceWhere", () => {
  it("reads enabled rows that are not deleted", () => {
    const where = liveWorkspaceWhere("ws_123", "acme");
    if (!where) {
      throw new Error("expected a workspace filter");
    }
    const rendered = JSON.stringify(where, (_key, value: unknown) => {
      if (value && typeof value === "object" && "name" in value && typeof value.name === "string") {
        return value.name;
      }
      return value;
    });
    expect(rendered).toContain("deleted_at_m");
    expect(rendered.toLowerCase()).toContain("is null");
    expect(rendered).toContain("enabled");
    expect(rendered).toContain("ws_123");
    expect(rendered).toContain("acme");
    expect(rendered).toContain("true");
  });

  it("requires workspaceId or slug", () => {
    expect(liveWorkspaceWhere()).toBeUndefined();
  });
});

describe("createAgentWorkspace", () => {
  it("creates another workspace with a new slug for the authorizing user", async () => {
    await expect(
      createAgentWorkspace({
        agent: { registrationId: "agent_reg_abc", userId: "user_existing" },
        name: "Acme Agent",
        slug: "acme-agent",
        audit: { location: "127.0.0.1" },
      }),
    ).resolves.toEqual({ orgId: "org_new", workspaceId: "ws_new", slug: "acme-agent" });
    expect(createFreeWorkspace).toHaveBeenCalledWith(
      expect.objectContaining({
        userId: "user_existing",
        slug: "acme-agent",
        name: "Acme Agent",
        metadata: { agent_registration_id: "agent_reg_abc" },
      }),
    );
  });
});

describe("issueAgentRootKey", () => {
  beforeEach(() => {
    state.selectCalls = 0;
    state.env.UNKEY_AGENT_SIGNUP_JWT_PRIVATE_KEY = undefined;
    state.env.UNKEY_AGENT_SIGNUP_JWT_SECRET = "local-dev-agent-signup-jwt-secret-32b";
  });

  it("mints through the agent route with the agent_signup role", async () => {
    const fetchImpl = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      expect(String(input)).toBe("https://api.unkey.test/v2/rootKeys.createAgentKey");
      const header = new Headers(init?.headers).get("authorization");
      const token = header?.replace("Bearer ", "") ?? "";
      const claims = decodeJwt(token);
      expect(claims.iss).toBe(AGENT_SIGNUP_ISSUER);
      expect(claims.aud).toBe(AGENT_SIGNUP_AUDIENCE);
      expect(claims.roles).toEqual([AGENT_SIGNUP_ROLE]);
      expect(claims).not.toMatchObject({ role: "admin" });
      expect(JSON.parse(String(init?.body))).toMatchObject({
        name: "Agent",
        permissions: ["unkey:v1:ws_123:projects/*/keyspaces/*#read"],
      });
      return Response.json({ data: { keyId: "key_123", key: "unkey_secret" } });
    });
    vi.stubGlobal("fetch", fetchImpl);

    await expect(
      issueAgentRootKey({
        agent: { registrationId: "agent_reg_abc", userId: "user_abc" },
        workspaceId: "ws_123",
        permissions: [{ path: "projects/*/keyspaces/*", action: "read" }],
      }),
    ).resolves.toMatchObject({ keyId: "key_123", key: "unkey_secret" });
    expect(state.selectCalls).toBe(1);
    expect(auth.listMemberships).toHaveBeenCalledWith("user_abc", "org_123");
  });

  it("mints a root key for a workspace the authorizing user already admins", async () => {
    const fetchImpl = vi.fn(async () =>
      Response.json({ data: { keyId: "key_existing", key: "unkey_existing" } }),
    );
    vi.stubGlobal("fetch", fetchImpl);

    await expect(
      issueAgentRootKey({
        agent: { registrationId: "agent_reg_abc", userId: "user_existing" },
        slug: "acme",
      }),
    ).resolves.toMatchObject({ keyId: "key_existing", key: "unkey_existing" });
    expect(auth.listMemberships).toHaveBeenCalledWith("user_existing", "org_123");
  });

  it("logs the upstream status and error fields when creation is rejected", async () => {
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        Response.json(
          {
            meta: { requestId: "req_abc" },
            error: {
              title: "Bad Request",
              type: "https://unkey.com/docs/errors/unkey/application/invalid_input",
              detail: "POST Path '/v2/rootKeys.createAgentKey' not found",
            },
          },
          { status: 400 },
        ),
      ),
    );

    await expect(
      issueAgentRootKey({
        agent: { registrationId: "agent_reg_abc", userId: "user_abc" },
        workspaceId: "ws_123",
      }),
    ).rejects.toMatchObject({ status: 502, code: "root_key_rejected" });

    const logged = JSON.stringify(errorSpy.mock.calls);
    expect(logged).toContain("400");
    expect(logged).toContain("req_abc");
    expect(logged).toContain("Bad Request");
    expect(logged).toContain("invalid_input");
    expect(logged).toContain("not found");
    expect(logged).not.toContain("Bearer");
    expect(logged).not.toContain("eyJ");
    expect(logged).not.toContain("local-dev-agent-signup");
    errorSpy.mockRestore();
  });

  it("logs the fetch failure class when the root key API cannot be reached", async () => {
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    vi.stubGlobal(
      "fetch",
      vi.fn(() => {
        throw new TypeError("fetch failed");
      }),
    );

    await expect(
      issueAgentRootKey({
        agent: { registrationId: "agent_reg_abc", userId: "user_abc" },
        workspaceId: "ws_123",
      }),
    ).rejects.toMatchObject({ status: 503, code: "upstream" });

    const logged = JSON.stringify(errorSpy.mock.calls);
    expect(logged).toContain("TypeError");
    expect(logged).not.toContain("Bearer");
    expect(logged).not.toContain("eyJ");
    errorSpy.mockRestore();
  });

  it("returns not_configured when signing material is missing", async () => {
    state.env.UNKEY_AGENT_SIGNUP_JWT_SECRET = "";
    state.env.UNKEY_AGENT_SIGNUP_JWT_PRIVATE_KEY = undefined;
    await expect(
      issueAgentRootKey({
        agent: { registrationId: "agent_reg_abc", userId: "user_abc" },
        slug: "acme",
      }),
    ).rejects.toMatchObject({ status: 503, code: "not_configured" });
  });
});

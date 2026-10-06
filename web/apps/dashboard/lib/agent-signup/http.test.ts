// @vitest-environment node
import { NextRequest } from "next/server";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/db", () => ({
  primaryDb: {},
}));

import { WorkspaceCreateError } from "@/lib/workspace/create-workspace";
import type { AgentSignupConfig } from "./config";
import { agentError, protectedResourceDocument, readRootKeyBody, readWorkspaceBody } from "./http";

const config: AgentSignupConfig = {
  apiKey: "sk_test",
  clientId: "client_123",
  issuer: "https://auth.example.com",
  audience: "client_123",
  apiBase: "https://api.workos.com",
  jwksUrl: "https://auth.example.com/oauth2/jwks",
};

const originalBase = process.env.DASHBOARD_BASE_URL;

afterEach(() => {
  if (originalBase === undefined) {
    delete process.env.DASHBOARD_BASE_URL;
  } else {
    process.env.DASHBOARD_BASE_URL = originalBase;
  }
});

function request(path: string, body: unknown) {
  return new NextRequest(`https://app.example.com${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

describe("protectedResourceDocument", () => {
  it("advertises the dashboard origin as the resource", () => {
    process.env.DASHBOARD_BASE_URL = "https://app.unkey.com";
    expect(protectedResourceDocument(config)).toEqual({
      resource: "https://app.unkey.com",
      authorization_servers: ["https://auth.example.com"],
    });
  });
});

describe("request bodies", () => {
  it("accepts a workspace name and slug", async () => {
    await expect(
      readWorkspaceBody(request("/api/agent/workspace", { name: "Acme", slug: "acme" })),
    ).resolves.toEqual({
      name: "Acme",
      slug: "acme",
    });
  });

  it("requires workspaceId or slug on a root key", async () => {
    await expect(
      readRootKeyBody(request("/api/agent/root-key", { name: "agent" })),
    ).rejects.toMatchObject({
      status: 400,
      code: "invalid_body",
    });
    await expect(
      readRootKeyBody(request("/api/agent/root-key", { slug: "acme" })),
    ).resolves.toMatchObject({ slug: "acme" });
  });
});

describe("agentError", () => {
  it("maps a taken slug to 409", async () => {
    const response = agentError(new WorkspaceCreateError("CONFLICT", "taken"));
    expect(response.status).toBe(409);
    await expect(response.json()).resolves.toMatchObject({ error: "slug_taken" });
  });
});

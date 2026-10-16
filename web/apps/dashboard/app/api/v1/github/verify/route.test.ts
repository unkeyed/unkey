import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  findKeys: vi.fn(),
  findWorkspaces: vi.fn(),
  getOrganizationMemberList: vi.fn(),
  sendLeakedKeyEmail: vi.fn(),
  captureException: vi.fn(),
  verifyGitSignature: vi.fn(),
  delivered: [] as string[],
}));

vi.mock("@/lib/env", () => ({
  env: () => ({
    RESEND_API_KEY: "re_test",
    GITHUB_KEYS_URI: "https://api.github.com/meta/public_keys/secret_scanning",
  }),
}));

vi.mock("@/lib/db", () => ({
  db: {
    query: {
      keys: { findMany: mocks.findKeys },
      workspaces: { findMany: mocks.findWorkspaces },
    },
  },
}));

vi.mock("@/lib/auth/server", () => ({
  auth: { getOrganizationMemberList: mocks.getOrganizationMemberList },
}));

vi.mock("@unkey/hash", () => ({
  sha256: async (source: string) => `hash:${source}`,
}));

vi.mock("@unkey/resend", () => ({
  Resend: class {
    async sendLeakedKeyEmail(email: { email: string }): Promise<void> {
      mocks.delivered.push(email.email);
      await mocks.sendLeakedKeyEmail(email);
    }
  },
}));

vi.mock("@sentry/nextjs", () => ({
  captureException: mocks.captureException,
}));

vi.mock("./verify-signature", () => ({
  verifyGitSignature: mocks.verifyGitSignature,
}));

import { POST } from "./route";

const REPORTED_TOKEN = {
  token: "unkey_live_root_key",
  source: "commit",
  url: "https://github.com/acme/repo/blob/abc/.env",
  type: "unkey_root_key",
};

function webhook(payload: unknown = [REPORTED_TOKEN]): Request {
  return new Request("https://app.unkey.com/api/v1/github/verify", {
    method: "POST",
    headers: {
      "github-public-key-signature": "signature",
      "github-public-key-identifier": "identifier",
    },
    body: JSON.stringify(payload),
  });
}

describe("POST /api/v1/github/verify", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.delivered.length = 0;
    mocks.verifyGitSignature.mockResolvedValue(true);
    mocks.findKeys.mockResolvedValue([
      { id: "key_1", hash: "hash:unkey_live_root_key", forWorkspaceId: "ws_1" },
    ]);
    mocks.findWorkspaces.mockResolvedValue([{ id: "ws_1", name: "acme", orgId: "org_1" }]);
    mocks.getOrganizationMemberList.mockResolvedValue({
      data: [{ user: { id: "user_1", firstName: "Ada", email: "ada@example.com" } }],
    });
    mocks.sendLeakedKeyEmail.mockResolvedValue(undefined);
    delete process.env.SLACK_WEBHOOK_URL_LEAKED_KEY;
  });

  it("delivers the leaked-key email before it answers GitHub", async () => {
    const response = await POST(webhook());

    expect(response.status).toBe(201);
    expect(mocks.delivered).toEqual(["ada@example.com"]);
    expect(mocks.sendLeakedKeyEmail).toHaveBeenCalledWith({
      email: "ada@example.com",
      date: expect.any(String),
      source: "commit",
      url: "https://github.com/acme/repo/blob/abc/.env",
      idempotencyKey: "leaked-key:hash:unkey_live_root_key:ada@example.com",
    });
  });

  it("answers GitHub without notifying when no reported token matches a live key", async () => {
    mocks.findKeys.mockResolvedValue([]);

    const response = await POST(webhook());

    expect(response.status).toBe(201);
    expect(mocks.delivered).toEqual([]);
  });

  it("still answers GitHub when the notification fails", async () => {
    mocks.sendLeakedKeyEmail.mockRejectedValue(new Error("resend unavailable"));

    const response = await POST(webhook());

    expect(response.status).toBe(201);
    expect(mocks.captureException).toHaveBeenCalled();
  });
});

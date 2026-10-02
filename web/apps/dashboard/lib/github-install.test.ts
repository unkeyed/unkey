// @vitest-environment node
import crypto from "node:crypto";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const fixture = vi.hoisted(() => ({
  configured: true,
  cookies: new Map<string, string>(),
  setCookie: vi.fn(),
}));

vi.mock("@/lib/env", () => ({
  githubAppEnv: () =>
    fixture.configured ? { GITHUB_APP_ID: 1, UNKEY_GITHUB_PRIVATE_KEY_PEM: "test-key" } : null,
  githubOAuthEnv: () =>
    fixture.configured ? { GITHUB_CLIENT_ID: "client", GITHUB_CLIENT_SECRET: "secret" } : null,
}));
vi.mock("next/headers", () => ({
  cookies: async () => ({
    get: (name: string) => ({ value: fixture.cookies.get(name) }),
    set: fixture.setCookie,
  }),
}));

import {
  finishGithubInstall,
  githubCallbackURL,
  githubInstallAvailable,
  githubRelayConfig,
  prepareGithubInstall,
  redeemGithubInstall,
} from "./github-install";

const transaction = "t".repeat(43);
const relayOrigin = "https://relay.example.com";
const sourceOrigin = "https://preview.example.com";
const cookieName = `__Host-github-install-${transaction}`;

describe("optional GitHub installation", () => {
  beforeEach(() => {
    fixture.configured = true;
    fixture.cookies.clear();
    fixture.setCookie.mockReset();
    fixture.setCookie.mockImplementation((name: string, value: string) =>
      fixture.cookies.set(name, value),
    );
    vi.stubEnv("NEXT_PUBLIC_GITHUB_APP_NAME", "test-app");
    vi.stubEnv("DASHBOARD_BASE_URL", sourceOrigin);
    vi.stubEnv("GITHUB_INSTALL_RELAY_URL", "");
    vi.stubEnv("GITHUB_INSTALL_RELAY_TOKEN", "");
    vi.stubEnv("GITHUB_INSTALL_RELAY_ADMIN_TOKEN", "");
    vi.stubGlobal("fetch", vi.fn());
  });
  afterEach(() => {
    vi.unstubAllEnvs();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("needs no relay or GitHub credentials when GitHub is disabled", async () => {
    fixture.configured = false;
    expect(githubRelayConfig()).toBeNull();
    expect(githubInstallAvailable()).toBe(false);
    await expect(prepareGithubInstall("state", "user", "workspace")).rejects.toMatchObject({
      code: "PRECONDITION_FAILED",
    });
    expect(fetch).not.toHaveBeenCalled();
    expect(fixture.setCookie).not.toHaveBeenCalled();
  });

  it("uses direct installation and the canonical callback without any relay calls", async () => {
    expect(githubInstallAvailable()).toBe(true);
    const url = new URL(await prepareGithubInstall("signed state & value", "user", "workspace"));
    expect(url.origin + url.pathname).toBe("https://github.com/apps/test-app/installations/new");
    expect(url.searchParams.get("state")).toBe("signed state & value");
    expect(githubCallbackURL()).toBe(`${sourceOrigin}/integrations/github/callback`);
    expect(fetch).not.toHaveBeenCalled();
    expect(fixture.setCookie).not.toHaveBeenCalled();
  });

  it("automatically enrolls the canonical origin and uses only its scoped token for transactions", async () => {
    vi.stubEnv("GITHUB_INSTALL_RELAY_URL", relayOrigin);
    vi.stubEnv("GITHUB_INSTALL_RELAY_ADMIN_TOKEN", "a".repeat(32));
    const now = Date.now();
    vi.spyOn(Date, "now").mockReturnValue(now);
    const enrollment = {
      ok: true,
      json: async () => ({
        id: "e".repeat(43),
        token: "s".repeat(43),
        origin: sourceOrigin,
        expiresAt: now + 86400_000,
      }),
    };
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(enrollment)
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          id: transaction,
          url: `${relayOrigin}/start?transaction=${transaction}`,
          expiresAt: now + 60_000,
        }),
      })
      .mockResolvedValueOnce(enrollment)
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          state: "signed-state",
          installationId: 42,
          origin: sourceOrigin,
        }),
      });
    vi.stubGlobal("fetch", fetchMock);
    expect(githubInstallAvailable()).toBe(true);
    expect(await prepareGithubInstall("signed-state", "alice", "workspace-a")).toBe(
      `${relayOrigin}/start?transaction=${transaction}`,
    );
    vi.resetModules();
    const restarted = await import("./github-install");
    await expect(
      restarted.redeemGithubInstall(transaction, "h".repeat(43), "alice", "workspace-a"),
    ).resolves.toMatchObject({ installationId: 42 });
    for (const call of [1, 3]) {
      expect(fetchMock).toHaveBeenNthCalledWith(
        call,
        `${relayOrigin}/v1/environments`,
        expect.objectContaining({
          headers: {
            Authorization: `Bearer ${"a".repeat(32)}`,
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            origin: sourceOrigin,
            expiresAt: now + 86400_000,
            reuse: true,
          }),
          redirect: "error",
          cache: "no-store",
        }),
      );
    }
    for (const [call, path] of [
      [2, "/v1/transactions"],
      [4, `/v1/transactions/${transaction}/redeem`],
    ] as const) {
      expect(fetchMock).toHaveBeenNthCalledWith(
        call,
        `${relayOrigin}${path}`,
        expect.objectContaining({
          headers: {
            Authorization: `Bearer ${"s".repeat(43)}`,
            "Content-Type": "application/json",
          },
        }),
      );
    }
    const binding = fixture.cookies.get(cookieName);
    expect(binding).toMatch(/^[A-Za-z0-9_-]{43}$/);
    expect(binding).not.toBe("s".repeat(43));
    expect(binding).not.toBe("a".repeat(32));
    expect(fetchMock).toHaveBeenCalledTimes(4);
  });

  it.each(["wrong-origin", "expired", "too-long", "bad-token", "rejected"])(
    "does not prepare a transaction after %s enrollment",
    async (failure) => {
      vi.stubEnv("GITHUB_INSTALL_RELAY_URL", relayOrigin);
      vi.stubEnv("GITHUB_INSTALL_RELAY_ADMIN_TOKEN", "a".repeat(32));
      const now = Date.now();
      vi.spyOn(Date, "now").mockReturnValue(now);
      vi.mocked(fetch).mockResolvedValue(
        Response.json(
          {
            id: "e".repeat(43),
            token: failure === "bad-token" ? "invalid" : "s".repeat(43),
            origin: failure === "wrong-origin" ? "https://sibling.example.com" : sourceOrigin,
            expiresAt:
              failure === "expired" ? now : now + (failure === "too-long" ? 31 : 1) * 86400_000,
          },
          { status: failure === "rejected" ? 400 : 200 },
        ),
      );
      await expect(prepareGithubInstall("state", "alice", "workspace-a")).rejects.toMatchObject({
        code: "BAD_REQUEST",
      });
      expect(fetch).toHaveBeenCalledTimes(1);
      expect(fixture.setCookie).not.toHaveBeenCalled();
    },
  );

  it.each([
    ["", "a".repeat(32), ""],
    [relayOrigin, "short", ""],
    [relayOrigin, "a".repeat(32), "s".repeat(43)],
  ])("rejects incomplete or ambiguous automatic enrollment settings", (url, adminToken, token) => {
    vi.stubEnv("GITHUB_INSTALL_RELAY_URL", url);
    vi.stubEnv("GITHUB_INSTALL_RELAY_ADMIN_TOKEN", adminToken);
    vi.stubEnv("GITHUB_INSTALL_RELAY_TOKEN", token);
    expect(githubInstallAvailable()).toBe(false);
    expect(fetch).not.toHaveBeenCalled();
  });

  it("never enrolls an unsafe dashboard origin", async () => {
    vi.stubEnv("GITHUB_INSTALL_RELAY_URL", relayOrigin);
    vi.stubEnv("GITHUB_INSTALL_RELAY_ADMIN_TOKEN", "a".repeat(32));
    vi.stubEnv("DASHBOARD_BASE_URL", "https://user@preview.example.com");
    await expect(prepareGithubInstall("state", "alice", "workspace-a")).rejects.toThrow();
    expect(fetch).not.toHaveBeenCalled();
  });

  it.each([
    [relayOrigin, ""],
    ["", "r".repeat(43)],
    ["http://relay.example.com", "r".repeat(43)],
    ["https://user@relay.example.com", "r".repeat(43)],
    ["https://*.example.com", "r".repeat(43)],
    [`${relayOrigin}/path`, "r".repeat(43)],
  ])("fails closed for partial or unsafe relay configuration %s", async (url, token) => {
    vi.stubEnv("GITHUB_INSTALL_RELAY_URL", url);
    vi.stubEnv("GITHUB_INSTALL_RELAY_TOKEN", token);
    expect(githubInstallAvailable()).toBe(false);
    await expect(prepareGithubInstall("state", "user", "workspace")).rejects.toMatchObject({
      code: "PRECONDITION_FAILED",
    });
    expect(fetch).not.toHaveBeenCalled();
  });

  it("keeps the browser binding out of URLs and redeems with the current user/workspace", async () => {
    vi.stubEnv("GITHUB_INSTALL_RELAY_URL", relayOrigin);
    vi.stubEnv("GITHUB_INSTALL_RELAY_TOKEN", "r".repeat(43));
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          id: transaction,
          url: `${relayOrigin}/start?transaction=${transaction}`,
          expiresAt: Date.now() + 60_000,
        }),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          state: "signed-state",
          installationId: 42,
          origin: sourceOrigin,
        }),
      });
    vi.stubGlobal("fetch", fetchMock);
    const url = await prepareGithubInstall("signed-state", "alice", "workspace-a");
    const binding = fixture.cookies.get(cookieName);
    if (!binding) {
      throw new Error("Missing binding cookie");
    }
    expect(url).toBe(`${relayOrigin}/start?transaction=${transaction}`);
    expect(url).not.toContain(binding);
    expect(fixture.setCookie).toHaveBeenCalledWith(
      cookieName,
      binding,
      expect.objectContaining({
        secure: true,
        httpOnly: true,
        sameSite: "lax",
        path: "/",
      }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      `${relayOrigin}/v1/transactions`,
      expect.objectContaining({
        redirect: "error",
        cache: "no-store",
        body: JSON.stringify({
          origin: sourceOrigin,
          state: "signed-state",
          userId: "alice",
          workspaceId: "workspace-a",
          binding: crypto.createHash("sha256").update(binding).digest("hex"),
        }),
      }),
    );
    await expect(
      redeemGithubInstall(transaction, "h".repeat(43), "alice", "workspace-a"),
    ).resolves.toMatchObject({ installationId: 42 });
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      `${relayOrigin}/v1/transactions/${transaction}/redeem`,
      expect.objectContaining({
        body: JSON.stringify({
          handoff: "h".repeat(43),
          binding,
          userId: "alice",
          workspaceId: "workspace-a",
        }),
      }),
    );
    await finishGithubInstall(transaction);
    await expect(
      redeemGithubInstall(transaction, "h".repeat(43), "alice", "workspace-a"),
    ).rejects.toMatchObject({ code: "BAD_REQUEST" });
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("rejects a relay response naming another source origin", async () => {
    vi.stubEnv("GITHUB_INSTALL_RELAY_URL", relayOrigin);
    vi.stubEnv("GITHUB_INSTALL_RELAY_TOKEN", "r".repeat(43));
    fixture.cookies.set(cookieName, "b".repeat(43));
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          state: "state",
          installationId: 42,
          origin: "https://other.example.com",
        }),
      }),
    );
    await expect(
      redeemGithubInstall(transaction, "h".repeat(43), "alice", "workspace-a"),
    ).rejects.toMatchObject({ code: "BAD_REQUEST" });
  });

  it("rejects arbitrary start destinations returned by a misconfigured relay", async () => {
    vi.stubEnv("GITHUB_INSTALL_RELAY_URL", relayOrigin);
    vi.stubEnv("GITHUB_INSTALL_RELAY_TOKEN", "r".repeat(43));
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          id: transaction,
          url: "https://other.example.com/start",
          expiresAt: Date.now() + 60_000,
        }),
      }),
    );
    await expect(prepareGithubInstall("state", "alice", "workspace-a")).rejects.toMatchObject({
      code: "BAD_REQUEST",
    });
    expect(fixture.setCookie).not.toHaveBeenCalled();
  });
});

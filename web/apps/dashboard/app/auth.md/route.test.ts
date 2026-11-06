import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  authkitUrl: "https://example.authkit.app" as string | undefined,
  fetch: vi.fn(),
}));

vi.mock("@/lib/env", () => ({
  env: () => ({ WORKOS_AUTHKIT_URL: mocks.authkitUrl, UNKEY_API_URL: "https://api.unkey.com" }),
}));

vi.mock("@/lib/utils", () => ({
  getBaseUrl: () => "https://app.unkey.com",
}));

import { GET } from "./route";

describe("GET /auth.md", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.authkitUrl = "https://example.authkit.app";
    vi.stubGlobal("fetch", mocks.fetch);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("is not found when agent registration is not configured", async () => {
    mocks.authkitUrl = undefined;

    const response = await GET();

    expect(response.status).toBe(404);
    expect(mocks.fetch).not.toHaveBeenCalled();
  });

  it("serves the hosted file with the Unkey section as markdown", async () => {
    mocks.fetch.mockResolvedValue(new Response("# auth.md\n"));

    const response = await GET();

    expect(response.status).toBe(200);
    expect(response.headers.get("Content-Type")).toBe("text/markdown; charset=utf-8");
    const body = await response.text();
    expect(body.startsWith("# auth.md\n\n## Use the token with Unkey\n")).toBe(true);
    expect(mocks.fetch.mock.calls[0][0].toString()).toBe(
      "https://example.authkit.app/agent/auth.md",
    );
  });

  it("reports the upstream failure when AuthKit answers with an error", async () => {
    vi.spyOn(console, "error").mockImplementation(() => {});
    mocks.fetch.mockResolvedValue(new Response("down", { status: 500 }));

    const response = await GET();

    expect(response.status).toBe(502);
    expect(await response.text()).toContain("Retry in a minute.");
  });

  it("reports the upstream failure when AuthKit cannot be reached", async () => {
    vi.spyOn(console, "error").mockImplementation(() => {});
    mocks.fetch.mockRejectedValue(new Error("timed out"));

    const response = await GET();

    expect(response.status).toBe(502);
  });
});

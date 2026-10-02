import { generateKeyPairSync } from "node:crypto";
import { afterEach, expect, it, vi } from "vitest";
import { z } from "zod";
import { githubAppEnv } from "./env";
import { getInstallationAccessToken } from "./github";

vi.mock("@/lib/env", () => ({
  githubAppEnv: vi.fn(),
  githubOAuthEnv: vi.fn(),
}));

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

it("leaves one minute of clock skew within GitHub's ten-minute JWT limit", async () => {
  const nowSeconds = 1_790_000_000;
  vi.spyOn(Date, "now").mockReturnValue(nowSeconds * 1000);
  const { privateKey } = generateKeyPairSync("rsa", { modulusLength: 2048 });
  vi.mocked(githubAppEnv).mockReturnValue({
    GITHUB_APP_ID: 1234,
    UNKEY_GITHUB_PRIVATE_KEY_PEM: privateKey.export({ type: "pkcs8", format: "pem" }).toString(),
  });
  const fetchMock = vi.fn<typeof fetch>().mockImplementation(async (_input, init) => {
    const authorization = new Headers(init?.headers).get("Authorization");
    expect(authorization).toMatch(/^Bearer /);
    const payload = authorization?.split(".")[1];
    if (!payload) {
      throw new Error("JWT payload missing");
    }
    const claims = z
      .object({ iat: z.number(), exp: z.number(), iss: z.number() })
      .parse(JSON.parse(Buffer.from(payload, "base64url").toString()));
    const githubNowSeconds = nowSeconds - 60;
    expect(claims.iat).toBeLessThanOrEqual(githubNowSeconds);
    expect(claims.exp).toBeLessThanOrEqual(githubNowSeconds + 600);
    expect(claims.exp).toBeGreaterThan(nowSeconds);
    expect(claims.iss).toBe(1234);
    return new Response(
      JSON.stringify({ token: "test-token", expires_at: "2026-10-02T01:00:00Z" }),
    );
  });
  vi.stubGlobal("fetch", fetchMock);

  await expect(getInstallationAccessToken(4321)).resolves.toMatchObject({ token: "test-token" });
  expect(fetchMock).toHaveBeenCalledOnce();
});

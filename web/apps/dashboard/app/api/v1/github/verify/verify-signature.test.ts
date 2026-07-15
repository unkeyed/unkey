import crypto from "node:crypto";
import { afterEach, describe, expect, it, vi } from "vitest";
import { verifyGitSignature } from "./verify-signature";

const GITHUB_KEYS_URI = "https://api.github.com/meta/public_keys/secret_scanning";

function makeKeyPair() {
  return crypto.generateKeyPairSync("ec", { namedCurve: "prime256v1" });
}

function sign(payload: string, privateKey: crypto.KeyObject): string {
  return crypto.createSign("SHA256").update(payload).sign(privateKey).toString("base64");
}

function pem(publicKey: crypto.KeyObject): string {
  const exported = publicKey.export({ type: "spki", format: "pem" });
  return typeof exported === "string" ? exported : exported.toString("utf-8");
}

function mockGithubKeys(keys: Array<{ key_identifier: string; key: string; is_current: boolean }>) {
  vi.spyOn(globalThis, "fetch").mockResolvedValue(
    new Response(JSON.stringify({ public_keys: keys }), { status: 200 }),
  );
}

describe("verifyGitSignature", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("returns true for a valid signature", async () => {
    const { privateKey, publicKey } = makeKeyPair();
    const payload = JSON.stringify([{ token: "x" }]);
    const signature = sign(payload, privateKey);
    const keyId = "kid-1";

    mockGithubKeys([{ key_identifier: keyId, key: pem(publicKey), is_current: true }]);

    await expect(verifyGitSignature(payload, signature, keyId, GITHUB_KEYS_URI)).resolves.toBe(
      true,
    );
  });

  it("returns false for a forged signature", async () => {
    const { privateKey: attackerKey } = makeKeyPair();
    const { publicKey: trustedKey } = makeKeyPair();
    const payload = JSON.stringify([{ token: "x" }]);
    const signature = sign(payload, attackerKey);
    const keyId = "kid-1";

    mockGithubKeys([{ key_identifier: keyId, key: pem(trustedKey), is_current: true }]);

    await expect(verifyGitSignature(payload, signature, keyId, GITHUB_KEYS_URI)).resolves.toBe(
      false,
    );
  });

  it("returns false when the signature is for a different payload", async () => {
    const { privateKey, publicKey } = makeKeyPair();
    const signature = sign("different payload", privateKey);
    const keyId = "kid-1";

    mockGithubKeys([{ key_identifier: keyId, key: pem(publicKey), is_current: true }]);

    await expect(
      verifyGitSignature("real payload", signature, keyId, GITHUB_KEYS_URI),
    ).resolves.toBe(false);
  });

  it("returns false when the keyId is unknown", async () => {
    const { privateKey, publicKey } = makeKeyPair();
    const payload = "p";
    const signature = sign(payload, privateKey);

    mockGithubKeys([{ key_identifier: "kid-known", key: pem(publicKey), is_current: true }]);

    await expect(
      verifyGitSignature(payload, signature, "kid-unknown", GITHUB_KEYS_URI),
    ).resolves.toBe(false);
  });

  it("returns false when the GitHub response shape is invalid", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ keys: [] }), { status: 200 }),
    );

    await expect(verifyGitSignature("p", "sig", "kid", GITHUB_KEYS_URI)).resolves.toBe(false);
  });

  it("returns false when GitHub responds with an error status", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response("err", { status: 500 }));

    await expect(verifyGitSignature("p", "sig", "kid", GITHUB_KEYS_URI)).resolves.toBe(false);
  });
});

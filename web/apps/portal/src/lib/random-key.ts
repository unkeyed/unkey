import type { Key } from "~/routes/dave-initial-design/-seed";

const ALPHA62 = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";

export function randomChars(len: number): string {
  const bytes = new Uint8Array(len);
  crypto.getRandomValues(bytes);
  let out = "";
  for (let i = 0; i < len; i += 1) {
    out += ALPHA62[bytes[i] % ALPHA62.length];
  }
  return out;
}

export function newPlaintext(): { plaintext: string; start: string } {
  const plaintext = `uk_live_${randomChars(28)}`;
  return { plaintext, start: plaintext.slice(0, 12) };
}

export function makeKey(input: { name: string | null; expiration: Date | undefined }): {
  key: Key;
  plaintext: string;
} {
  const { plaintext, start } = newPlaintext();
  const key: Key = {
    id: `key_${randomChars(12)}`,
    name: input.name,
    start,
    createdAt: Date.now(),
    expires: input.expiration?.getTime() ?? null,
    enabled: true,
    usage: [],
  };
  return { key, plaintext };
}

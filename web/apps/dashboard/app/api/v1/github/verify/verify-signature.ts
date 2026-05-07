import { Buffer } from "node:buffer";
import crypto from "node:crypto";
import { z } from "zod";

const githubKeysSchema = z.object({
  public_keys: z.array(
    z.object({
      key_identifier: z.string(),
      key: z.string(),
      is_current: z.boolean(),
    }),
  ),
});

export async function verifyGitSignature(
  payload: string,
  signature: string,
  keyId: string,
  githubKeysUri: string,
): Promise<boolean> {
  const response = await fetch(githubKeysUri);
  if (!response.ok) {
    console.error("Github verify error", response.status, await response.text());
    return false;
  }

  const parsed = githubKeysSchema.safeParse(await response.json());
  if (!parsed.success) {
    console.error("Github keys response did not match expected shape", parsed.error.message);
    return false;
  }

  const publicKey = parsed.data.public_keys.find((k) => k.key_identifier === keyId);
  if (!publicKey) {
    console.error("No public key found");
    return false;
  }

  const verifier = crypto.createVerify("SHA256").update(payload);
  return verifier.verify(publicKey.key, Buffer.from(signature, "base64"));
}

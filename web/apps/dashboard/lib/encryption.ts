import {
  createHmac,
  createCipheriv,
  createDecipheriv,
  randomBytes,
} from "crypto";

const ALGORITHM = "aes-256-gcm";
const NONCE_LENGTH = 12;

/**
 * Derive a workspace-specific encryption key from the master key using HMAC-SHA256.
 * This matches the Go implementation in pkg/encryption/workspace.go
 * Note: masterKey is decoded from base64 before use
 */
function deriveWorkspaceKey(masterKeyB64: string, workspaceId: string): Buffer {
  // Decode master key from base64
  const masterKeyBytes = Buffer.from(masterKeyB64, "base64");
  const h = createHmac("sha256", masterKeyBytes);
  h.update("unkey-billing-encryption:");
  h.update(workspaceId);
  return h.digest();
}

/**
 * Encrypt data using AES-256-GCM.
 * This matches the Go implementation in pkg/encryption/workspace.go
 */
function encrypt(
  key: Buffer,
  plaintext: Buffer
): { nonce: Buffer; ciphertext: Buffer } {
  const nonce = randomBytes(NONCE_LENGTH);
  const cipher = createCipheriv(ALGORITHM, key, nonce);

  let ciphertext = cipher.update(plaintext);
  ciphertext = Buffer.concat([ciphertext, cipher.final()]);

  const authTag = cipher.getAuthTag();

  // Concatenate ciphertext and auth tag (GCM mode requires auth tag for decryption)
  return { nonce, ciphertext: Buffer.concat([ciphertext, authTag]) };
}

/**
 * Decrypt data using AES-256-GCM.
 * This matches the Go implementation in pkg/encryption/workspace.go
 */
function decrypt(
  key: Buffer,
  nonce: Buffer,
  data: Buffer
): Buffer {
  const decipher = createDecipheriv(ALGORITHM, key, nonce);

  // Extract auth tag from the end of the ciphertext (16 bytes for GCM)
  const authTagLength = 16;
  const ciphertext = data.subarray(0, data.length - authTagLength);
  const authTag = data.subarray(data.length - authTagLength);

  decipher.setAuthTag(authTag);

  let plaintext = decipher.update(ciphertext);
  plaintext = Buffer.concat([plaintext, decipher.final()]);

  return plaintext;
}

/**
 * Encrypt a token for a specific workspace.
 * Returns base64-encoded nonce:ciphertext format.
 * This matches the Go implementation in pkg/encryption/workspace.go
 */
export function encryptToken(
  workspaceId: string,
  token: string,
  masterKey: string
): string {
  if (token === "") {
    throw new Error("token cannot be empty");
  }

  const workspaceKey = deriveWorkspaceKey(masterKey, workspaceId);
  const { nonce, ciphertext } = encrypt(workspaceKey, Buffer.from(token));

  // Format: base64(nonce):base64(ciphertext)
  return `${nonce.toString("base64")}:${ciphertext.toString("base64")}`;
}

/**
 * Decrypt a token for a specific workspace.
 * Expects base64-encoded nonce:ciphertext format.
 * This matches the Go implementation in pkg/encryption/workspace.go
 */
export function decryptToken(
  workspaceId: string,
  encrypted: string,
  masterKey: string
): string {
  if (encrypted === "") {
    throw new Error("encrypted token cannot be empty");
  }

  const workspaceKey = deriveWorkspaceKey(masterKey, workspaceId);

  // Split the encoded string into nonce and ciphertext
  const separatorIndex = encrypted.indexOf(":");
  if (separatorIndex === -1) {
    throw new Error("invalid encrypted token format: missing separator");
  }

  const nonce = Buffer.from(encrypted.substring(0, separatorIndex), "base64");
  const ciphertext = Buffer.from(encrypted.substring(separatorIndex + 1), "base64");

  const plaintext = decrypt(workspaceKey, nonce, ciphertext);
  return plaintext.toString("utf-8");
}
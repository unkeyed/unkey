import { encryptToken, decryptToken } from "./web/apps/dashboard/lib/encryption";

// Test encryption/decryption
const masterKey = "02a7a5b6c686c6c6f776f726c64";
const workspaceId = "ws_5ZUapZkYqhPpbkfL";
const testToken = "test_access_token_12345";

console.log("Testing encryption...");
console.log("MASTER_KEY length:", masterKey.length, "bytes");
console.log("MASTER_KEY bytes:", Buffer.from(masterKey, "utf-8").toString("hex"));

// Encrypt
const encrypted = encryptToken(workspaceId, testToken, masterKey);
console.log("Encrypted:", encrypted);

// Decrypt
const decrypted = decryptToken(workspaceId, encrypted, masterKey);
console.log("Decrypted:", decrypted);

// Verify
if (decrypted === testToken) {
  console.log("✓ SUCCESS: Encryption/decryption works!");
} else {
  console.log("✗ FAILURE: Decrypted token doesn't match");
  console.log("Expected:", testToken);
  console.log("Got:", decrypted);
}
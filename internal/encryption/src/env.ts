type Env = {
  ENCRYPTION_KEYS: { version: number; key: string }[];
};
/**
 * Returns the key with the highest version indicating it is the latest and should be used to encrypt
 * data
 */
export function getEncryptionKeyFromEnv(env: Env): { key: string; version: number } {
  let latest = env.ENCRYPTION_KEYS[0];
  for (let i = 1; i < env.ENCRYPTION_KEYS.length; i++) {
    if (env.ENCRYPTION_KEYS[i].version > latest.version) {
      latest = env.ENCRYPTION_KEYS[i];
    }
  }
  return latest;
}

/**
 * Returns the key with the highest version indicating it is the latest and should be used to encrypt
 * data
 */
export function getDecryptionKeyFromEnv(env: Env, version: number): string {
  for (const key of env.ENCRYPTION_KEYS) {
    if (key.version === version) {
      return key.key;
    }
  }
  throw new Error(`No decryption key found for version ${version}`);
}

import { EnvError, Err, Ok, type Result } from "@unkey/error";
export type Env = {
  ENCRYPTION_KEYS: { version: number; key: string }[];
};

/**
 * Returns the key with the highest version indicating it is the latest and should be used to encrypt
 * data
 */
export function getEncryptionKeyFromEnv(
  env: Env,
): Result<{ key: string; version: number }, EnvError> {
  if (env.ENCRYPTION_KEYS.length === 0) {
    return Err(
      new EnvError({
        message: "encryption key array is empty",
        context: { name: "ENCRYPTION_KEYS" },
      }),
    );
  }
  let latest = env.ENCRYPTION_KEYS[0];
  for (let i = 1; i < env.ENCRYPTION_KEYS.length; i++) {
    if (env.ENCRYPTION_KEYS[i].version > latest.version) {
      latest = env.ENCRYPTION_KEYS[i];
    }
  }
  return Ok(latest);
}

/**
 * Returns the key with the highest version indicating it is the latest and should be used to encrypt
 * data
 */
export function getDecryptionKeyFromEnv(env: Env, version: number): Result<string, EnvError> {
  for (const key of env.ENCRYPTION_KEYS) {
    if (key.version === version) {
      return Ok(key.key);
    }
  }
  return Err(
    new EnvError({
      message: `No decryption key found for version ${version}`,
      context: { name: "ENCRYPTION_KEYS" },
    }),
  );
}

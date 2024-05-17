import { randomBytes, randomUUID } from "node:crypto";
import { execPath } from "node:process";
import { base64 } from "@unkey/encoding";
import { assert, describe, expect, test } from "vitest";
import { AesGCM } from "./aes-gcm";
import { getDecryptionKeyFromEnv, getEncryptionKeyFromEnv } from "./env";

describe("rolling encryption keys", () => {
  test("all values are correctly migrated", async () => {
    const currentKey = {
      key: base64.encode(randomBytes(32)),
      version: 1,
    };
    const nextKey = {
      key: base64.encode(randomBytes(32)),
      version: 2,
    };

    const current = await AesGCM.withBase64Key(currentKey.key);
    const next = await AesGCM.withBase64Key(nextKey.key);

    const secret = randomUUID();

    const encrypted = await current.encrypt(secret);
    const recovered = await current.decrypt(encrypted);

    expect(recovered).toEqual(secret);

    const migrated = await next.encrypt(recovered);

    expect(await next.decrypt(migrated)).toEqual(secret);

    expect(() => current.decrypt(migrated)).rejects.toThrowError(
      "The operation failed for an operation-specific reason",
    );
  });

  test("with env shifts", async () => {
    const currentKey = {
      key: base64.encode(randomBytes(32)),
      version: 1,
    };
    const nextKey = {
      key: base64.encode(randomBytes(32)),
      version: 2,
    };

    const secret = randomUUID();

    /**
     * STEP 1 - PRE MIGRATION
     */
    const env1 = { ENCRYPTION_KEYS: [currentKey] };

    const encryptionKey1 = getEncryptionKeyFromEnv(env1);
    assert(encryptionKey1.err === undefined);

    const aes1Enc = await AesGCM.withBase64Key(encryptionKey1.val.key);

    const encrypted1 = await aes1Enc.encrypt(secret);

    /**
     * STEP 2 - DURING MIGRATION
     */
    const env2 = { ENCRYPTION_KEYS: [currentKey, nextKey] };
    const decryptionKey1 = getDecryptionKeyFromEnv(env2, 1);
    assert(decryptionKey1.err === undefined);
    const aes1Dec = await AesGCM.withBase64Key(decryptionKey1.val);

    const recovered = await aes1Dec.decrypt(encrypted1);

    expect(recovered).toEqual(secret);
    const encryptionKey2 = getEncryptionKeyFromEnv(env2);
    assert(encryptionKey2.err === undefined);

    const aes2Enc = await AesGCM.withBase64Key(encryptionKey2.val.key);

    const migrated = await aes2Enc.encrypt(recovered);

    /**
     * STEP 3 - AFTER MIGRATION
     */
    const env3 = { ENCRYPTION_KEYS: [currentKey, nextKey] };
    const decryptionKey2 = getDecryptionKeyFromEnv(env3, 2);
    assert(decryptionKey2.err === undefined);
    const aes2Dec = await AesGCM.withBase64Key(decryptionKey2.val);

    expect(await aes2Dec.decrypt(migrated)).toEqual(secret);

    /**
     * STEP 4 - AFTER REMOVING currentKey
     */
    const env4 = { ENCRYPTION_KEYS: [nextKey] };
    const decryptionKey3 = getDecryptionKeyFromEnv(env4, 2);
    assert(decryptionKey3.err === undefined);
    const aes3Dec = await AesGCM.withBase64Key(decryptionKey3.val);

    expect(await aes3Dec.decrypt(migrated)).toEqual(secret);
  });
});

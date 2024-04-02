import { describe, expect, test } from "vitest";
import { getDecryptionKeyFromEnv, getEncryptionKeyFromEnv } from "./env";

describe("getEncryptionKeyFromEnv", () => {
  test("return the key with highest version", () => {
    const ENCRYPTION_KEYS = [
      {
        key: "key1",
        version: 1,
      },
      {
        key: "key3",
        version: 3,
      },
      {
        key: "key2",
        version: 2,
      },
    ];

    const res = getEncryptionKeyFromEnv({ ENCRYPTION_KEYS });

    expect(res.err).toBeUndefined();
    expect(res.val!.key).toEqual("key3");
    expect(res.val!.version).toEqual(3);
  });
});

describe("getDecryptionKeyFromEnv", () => {
  test("return the key with specified version", () => {
    const ENCRYPTION_KEYS = [
      {
        key: "key1",
        version: 1,
      },
      {
        key: "key3",
        version: 3,
      },
      {
        key: "key2",
        version: 2,
      },
    ];

    const key = getDecryptionKeyFromEnv({ ENCRYPTION_KEYS }, 2);
    expect(key.err).toBeUndefined();
    expect(key.val!).toEqual("key2");
  });
});

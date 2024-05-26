import { beforeEach, describe, test } from "vitest";
import type { Store } from "../stores";
import { MemoryStore } from "../stores/memory";
import { EncryptedStore } from "./encryption";

test("encrypts and decrypts", async (t) => {
  const map = new Map();
  const memory: Store<"namespace", string> = new MemoryStore({ persistentMap: map });

  // generated with `openssl rand -base64 32`
  const encryptionKey = "gVXaB49mnCZILHqXNpZ/cH22TsYoM5QbzjX3Nu15lKo=";
  const store = await EncryptedStore.withBase64Key(memory, encryptionKey);

  const key = "key";
  const value = "value";

  await store.set("namespace", key, {
    value,
    freshUntil: Date.now() + 1000000,
    staleUntil: Date.now() + 100000000,
  });

  const res = await store.get("namespace", key);
  t.expect(res.err).not.toBeDefined();
  t.expect(res.val!.value).toEqual(value);
});

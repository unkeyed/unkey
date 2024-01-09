"use server";
import { auth } from "@/auth";
import { Unkey } from "@unkey/api";

export async function createKey(amount: number) {
  const sess = await auth();
  const ownerId = sess?.user?.id;
  if (!ownerId) {
    throw new Error("No owner id");
  }
  const unkey = new Unkey({ rootKey: process.env.UNKEY_ROOT_KEY! });

  const key = await unkey.keys.create({
    apiId: process.env.UNKEY_API_ID!,
    ownerId,
    remaining: amount / 100,
  });

  return { key: key.result?.key, keyId: key.result?.keyId };
}

export async function updateKey(key: any, amount: number) {
  const sess = await auth();
  const ownerId = sess?.user?.id;
  if (!ownerId) {
    throw new Error("No owner id");
  }
  const unkey = new Unkey({ rootKey: process.env.UNKEY_ROOT_KEY! });

  await unkey.keys.updateRemaining({
    keyId: key.id,
    op: "increment",
    value: amount / 100,
  });
}

export async function listKeys() {
  const sess = await auth();
  const ownerId = sess?.user?.id;
  const unkey = new Unkey({ rootKey: process.env.UNKEY_ROOT_KEY! });

  const keys = await unkey.apis.listKeys({
    apiId: process.env.UNKEY_API_ID!,
    ownerId,
  });

  if (keys.error) {
    throw new Error(keys.error.message);
  }

  return keys.result.keys;
}

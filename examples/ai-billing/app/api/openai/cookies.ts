import { Unkey } from "@unkey/api";
import { cookies } from "next/headers";

export function readKeyFromCookie(cookieName: string) {
  const cookieStore = cookies();
  const cookie = cookieStore.get(cookieName)?.value as string;
  if (!cookie) {
    return null;
  }
  return cookie;
}

export async function resetCookieIfDeleted({ ownerId }: { ownerId: string }) {
  const unkey = new Unkey({ rootKey: process.env.UNKEY_ROOT_KEY! });

  const { error, result } = await unkey.apis.listKeys({
    apiId: process.env.UNKEY_API_ID!,
    ownerId,
  });

  if (error) {
    throw new Error("Error reading from Unkey: check your API ID");
  }

  const keys = result?.keys;

  if (keys.length) {
    const key = keys[0];
    const remaining = key.remaining;

    const newKey = await unkey.keys.create({
      apiId: process.env.UNKEY_API_ID!,
      ownerId,
      remaining,
    });

    if (newKey.error) {
      throw new Error("Error creating new key");
    }

    const newCookie = { key: newKey.result?.key, keyId: newKey.result?.keyId };

    await unkey.keys.delete({
      keyId: key.id,
    });

    return newCookie;
  } else {
    return null;
  }
}

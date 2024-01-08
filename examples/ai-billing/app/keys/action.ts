"use server";
import { auth } from "@/auth";
import { Unkey } from "@unkey/api";
import { revalidatePath } from "next/cache";

export async function createKey() {
  const sess = await auth();
  const ownerId = sess?.user?.id;
  if (!ownerId) {
    throw new Error("No owner id");
  }
  const unkey = new Unkey({ rootKey: process.env.UNKEY_ROOT_KEY! });

  const key = await unkey.keys.create({
    apiId: process.env.UNKEY_API_ID!,
    ownerId,
    remaining: 10,
  });
  return key.result?.key;
}

export async function revokeKey(formData: FormData) {
  const keyId = formData.get("keyId") as string;
  const sess = await auth();
  const ownerId = sess?.user?.id ?? sess?.user?.email;
  if (!ownerId) {
    throw new Error("No owner id");
  }

  const unkey = new Unkey({ rootKey: process.env.UNKEY_ROOT_KEY! });

  const { error, result } = await unkey.keys.get({
    keyId,
  });
  if (error) {
    throw new Error(error.message);
  }
  if (!result) {
    throw new Error("No key found");
  }
  if (result.ownerId !== ownerId) {
    throw new Error("Not authorized");
  }

  await unkey.keys.delete({ keyId });
  revalidatePath("/keys");
}

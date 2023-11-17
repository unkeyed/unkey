"use server";

import { Unkey } from "@unkey/api";
import { revalidatePath } from "next/cache";

export async function createKey(remaining: number) {
  const rootKey = process.env.UNKEY_ROOT_KEY;
  if (!rootKey) {
    throw new Error("UNKEY_ROOT_KEY is not defined");
  }
  const unkey = new Unkey({ rootKey });

  const apiId = process.env.UNKEY_API_ID;
  if (!apiId) {
    throw new Error("UNKEY_API_ID is not defined");
  }

  const res = await unkey.keys.create({
    apiId,
    prefix: "remaining",
    remaining: Number(remaining),
  });

  if (res.error) {
    console.error(res.error);
    throw new Error(res.error.message);
  }

  return revalidatePath("/");
}

export async function getKeys() {
  const rootKey = process.env.UNKEY_ROOT_KEY;
  if (!rootKey) {
    throw new Error("UNKEY_ROOT_KEY is not defined");
  }
  const unkey = new Unkey({ rootKey });

  const apiId = process.env.UNKEY_API_ID;
  if (!apiId) {
    throw new Error("UNKEY_API_ID is not defined");
  }

  const { result, error } = await unkey.apis.listKeys({ apiId });

  if (error) {
    console.error(error);
    throw new Error(error.message);
  }

  return { keys: result.keys };
}

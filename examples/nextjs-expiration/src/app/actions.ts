"use server";

import { keys } from "@/server/keys";
import { Unkey } from "@unkey/api";
import { revalidatePath } from "next/cache";

export async function createKey(_formDate: FormData) {
  const rootKey = process.env.UNKEY_ROOT_KEY;
  if (!rootKey) {
    throw new Error("UNKEY_ROOT_KEY is not defined");
  }
  const unkey = new Unkey({ rootKey });

  const apiId = process.env.UNKEY_API_ID;
  if (!apiId) {
    throw new Error("UNKEY_API_ID is not defined");
  }
  const expires = new Date().getTime() + 1000 * 60;
  const res = await unkey.keys.create({
    apiId,
    prefix: "forecast",
    expires,
  });

  if (res.error) {
    console.error(res.error);
    throw new Error(res.error.message);
  }

  keys.push({
    key: res.result.key,
    keyId: res.result.keyId,
    expires,
  });
  revalidatePath("/");
}

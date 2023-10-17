"use server";

import { revalidatePath } from "next/cache";
import { Unkey } from "@unkey/api";
import { keys } from "@/server/keys";

const UNKEY_ROOT_KEY = process.env.UNKEY_ROOT_KEY;
if (!UNKEY_ROOT_KEY) {
  throw new Error("UNKEY_ROOT_KEY is not defined");
}

function getApiId() {
  const UNKEY_API_ID = process.env.UNKEY_API_ID;
  if (!UNKEY_API_ID) {
    throw new Error("UNKEY_API_ID is not defined");
  }

  return UNKEY_API_ID;
}

const unkey = new Unkey({ token: UNKEY_ROOT_KEY });

export async function createKey(_formDate: FormData) {
  const expires = new Date().getTime() + 1000 * 60;
  const res = await unkey.keys.create({
    apiId: getApiId(),
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

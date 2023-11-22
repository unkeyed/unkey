"use server";

import { Unkey } from "@unkey/api";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
const loginAction = async () => {
  const rk = process.env.UNKEY_ROOT_KEY;
  const apiId = process.env.UNKEY_API_ID;

  if (!rk || !apiId) {
    return { message: "No root key or API ID specified" };
  }
  const unkey = new Unkey({
    rootKey: rk,
  });
  const created = await unkey.keys.create({
    apiId,
    ratelimit: {
      type: "fast",
      limit: 5,
      refillRate: 1,
      refillInterval: 1000,
    },
  });
  if (created.error) {
    return { message: "Unable to create API key" };
  }
  cookies().set("unkey-token", created.result.key);
  redirect("/");
};
export { loginAction };

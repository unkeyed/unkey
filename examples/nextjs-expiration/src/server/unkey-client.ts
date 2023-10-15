import { Unkey } from "@unkey/api";

type Key = {
  key: string;
  keyId: string;
  expires: number;
};

export const keys: Key[] = [];

const UNKEY_ROOT_KEY = process.env.UNKEY_ROOT_KEY;
if (!UNKEY_ROOT_KEY) throw new Error("UNKEY_ROOT_KEY is not defined");

function getApiId() {
  const UNKEY_API_ID = process.env.UNKEY_API_ID;
  if (!UNKEY_API_ID) throw new Error("UNKEY_API_ID is not defined");

  return UNKEY_API_ID;
}

export const unkey = new Unkey({ token: UNKEY_ROOT_KEY });

export async function createApiKey(args: { expires: number }) {
  const res = await unkey.keys.create({
    apiId: getApiId(),
    prefix: "forecast",
    expires: args.expires,
  });

  if (res.error) {
    console.error(res.error);
    throw new Error(res.error.message);
  }

  keys.push({
    key: res.result.key,
    keyId: res.result.keyId,
    expires: args.expires,
  });

  return res;
}

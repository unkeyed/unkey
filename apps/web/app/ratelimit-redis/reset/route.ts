import { cookies } from "next/headers";
export const runtime = "edge";

const UNKEY_RATELIMIT_COOKIE = "UNKEY_RATELIMIT_REDIS";

export const POST = async (_req: Request): Promise<Response> => {
  cookies().delete(UNKEY_RATELIMIT_COOKIE);
  return new Response("ok");
};

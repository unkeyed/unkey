import { cookies } from "next/headers";

const UNKEY_RATELIMIT_COOKIE = "UNKEY_RATELIMIT";

export const POST = async (_req: Request): Promise<Response> => {
  cookies().delete(UNKEY_RATELIMIT_COOKIE);
  return new Response("ok");
};

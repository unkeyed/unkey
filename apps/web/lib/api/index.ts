import { Unkey } from "@unkey/api";
import { env } from "../env";

export const unkeyScoped = (token: string) => new Unkey({ token, baseUrl: env.UNKEY_API_URL });
export const unkeyRoot = unkeyScoped(env.UNKEY_APP_AUTH_TOKEN);

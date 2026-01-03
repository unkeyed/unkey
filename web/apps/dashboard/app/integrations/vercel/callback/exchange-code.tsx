import { vercelIntegrationEnv } from "@/lib/env";
import { BaseError, Err, FetchError, Ok, type Result, SchemaError } from "@unkey/error";
import { z } from "zod";
export class VercelCodeExchangeError extends BaseError<{ status: number }> {
  public readonly retry = true;
  public readonly name = VercelCodeExchangeError.name;
}

export async function exchangeCode(code: string): Promise<
  Result<
    {
      accessToken: string;
      installationId: string;
      userId: string;
      teamId: string | null;
    },
    VercelCodeExchangeError | SchemaError | FetchError
  >
> {
  const url = "https://api.vercel.com/v2/oauth/access_token";
  const method = "POST";

  const env = vercelIntegrationEnv();
  if (!env?.VERCEL_INTEGRATION_CLIENT_ID || !env?.VERCEL_INTEGRATION_CLIENT_SECRET) {
    throw new Error(
      "Vercel integration environment variables are not configured. Check VERCEL_INTEGRATION_CLIENT_ID and VERCEL_INTEGRATION_CLIENT_SECRET.",
    );
  }

  const res = await fetch(url, {
    method,
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
    },
    body: new URLSearchParams({
      client_id: env.VERCEL_INTEGRATION_CLIENT_ID,
      client_secret: env.VERCEL_INTEGRATION_CLIENT_SECRET,
      code,
      redirect_uri: "http://localhost:3000/integrations/vercel/callback",
    }),
  }).catch((e) => {
    return new FetchError({
      message: (e as Error).message,
      retry: true,
      context: {
        url,
        method,
      },
    });
  });
  if (res instanceof FetchError) {
    return Err(res);
  }
  if (!res.ok) {
    return Err(
      new VercelCodeExchangeError({
        message: "failed to exchange code for access token",
        context: { status: res.status },
      }),
    );
  }
  const json = await res.json();
  const data = z
    .object({
      token_type: z.literal("Bearer"),
      access_token: z.string(),
      installation_id: z.string(),
      user_id: z.string(),
      team_id: z.string().nullable(),
    })
    .safeParse(json);

  if (!data.success) {
    return Err(SchemaError.fromZod(data.error, json));
  }
  return Ok({
    accessToken: data.data.access_token,
    installationId: data.data.installation_id,
    userId: data.data.user_id,
    teamId: data.data.team_id,
  });
}

import { vercelIntegrationEnv } from "@/lib/env";
import { BaseError, Err, FetchError, Ok, type Result, SchemaError } from "@unkey/error";
import { z } from "zod";
export class VercelCodeExchangeError extends BaseError<{ status: number }> {
  public readonly type = "VercelCodeExchangeError";
  public readonly retry = true;
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
  const res = await fetch(url, {
    method,
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
    },
    body: new URLSearchParams({
      client_id: vercelIntegrationEnv()!.VERCEL_INTEGRATION_CLIENT_ID,
      client_secret: vercelIntegrationEnv()!.VERCEL_INTEGRATION_CLIENT_SECRET,
      code,
      redirect_uri: "http://localhost:3000/integrations/vercel/callback",
    }),
  }).catch((e) => {
    return new FetchError((e as Error).message, {
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
      new VercelCodeExchangeError("failed to exchange code for access token", {
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

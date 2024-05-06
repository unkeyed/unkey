import { z } from "zod";

export function env() {
  const parsed = z
    .object({
      DATABASE_HOST: z.string(),
      DATABASE_USERNAME: z.string(),
      DATABASE_PASSWORD: z.string(),

      CLOUDFLARE_API_KEY: z.string(),
    })
    .safeParse(process.env);
  if (!parsed.success) {
    throw new Error(`env: ${parsed.error.message}`);
  }
  return parsed.data;
}

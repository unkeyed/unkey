import { z } from "zod";

export function env() {
  const parsed = z
    .object({
      FIRECRAWL_API_KEY: z.string(),
      SERPER_API_KEY: z.string(),
    })
    .safeParse(process.env);
  if (!parsed.success) {
    throw new Error(`env: ${parsed.error.message}`);
  }
  return parsed.data;
}

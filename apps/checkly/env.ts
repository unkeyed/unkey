import { z } from "zod";

const validation = z.object({
  UNKEY: z.string(),
  UNKEY_BASE_URL: z.string().default("https://api.unkey.dev"),
  UNKEY_API_ID: z.string(),
});

export const env = validation.parse(process.env);

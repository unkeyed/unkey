import { z } from "zod";

export const env = () =>
  z
    .object({
      DATABASE_HOST: z.string(),
      DATABASE_USERNAME: z.string(),
      DATABASE_PASSWORD: z.string(),
      UNKEY_API_URL: z.string().default("https://api.unkey.dev"),
    })
    .parse(process.env);

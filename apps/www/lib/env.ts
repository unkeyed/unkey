import { z } from "zod";

export const env = () =>
  z
    .object({
      NEXT_PUBLIC_BASE_URL: z.string().url().default("https://unkey.com"),
    })
    .parse(process.env);

export const dbEnv = () =>
  z
    .object({
      DATABASE_HOST: z.string(),
      DATABASE_USERNAME: z.string(),
      DATABASE_PASSWORD: z.string(),
    })
    .parse(process.env);

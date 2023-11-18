import { z } from "zod";

const requiredEnv = z.object({
  UNKEY_BASE_URL: z.string().url().default("http://127.0.0.1:8787"),
  UNKEY_ROOT_KEY: z.string(),
  DATABASE_HOST: z.string(),
  DATABASE_USERNAME: z.string(),
  DATABASE_PASSWORD: z.string(),
});

export function testEnv() {
  const res = requiredEnv.safeParse(process.env);
  if (!res.success) {
    throw new Error(`Missing required environment variables: ${res.error.message}`);
  }
  return res.data;
}

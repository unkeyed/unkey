import { z } from "zod";

const schema = z.object({
  VERCEL_ENV: z.enum(["development", "preview", "production"]).optional().default("development"),
  VERCEL_URL: z.string().optional(),
  DATABASE_HOST: z.string(),
  DATABASE_USERNAME: z.string(),
  DATABASE_PASSWORD: z.string(),

  UNKEY_WORKSPACE_ID: z.string(),
  UNKEY_API_ID: z.string(),

  UPSTASH_KAFKA_REST_URL: z.string(),
  UPSTASH_KAFKA_REST_USERNAME: z.string(),
  UPSTASH_KAFKA_REST_PASSWORD: z.string(),
});

export const env = schema.parse(process.env);

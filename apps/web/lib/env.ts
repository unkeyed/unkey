import { z } from "zod";

const schema = z.object({
  VERCEL_ENV: z.enum(["development", "preview", "production"]).optional().default("development"),
  VERCEL_URL: z.string().optional(),
  DATABASE_URL: z.string(),
});

export const env = schema.parse(process.env);

import { z } from "zod";

const schema = z.object({
  DATABASE_HOST: z.string(),
  DATABASE_USERNAME: z.string(),
  DATABASE_PASSWORD: z.string(),
  REDIS_URL: z.string(),
  TINYBIRD_TOKEN: z.string(),
});

export const env = schema.parse(process.env);

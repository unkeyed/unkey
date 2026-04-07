import { z } from "zod";

const envSchema = z.object({
  UNKEY_API_URL: z.string().default("https://api.unkey.dev"),
});

let _env: z.infer<typeof envSchema> | null = null;

export function env() {
  if (!_env) {
    _env = envSchema.parse(process.env);
  }
  return _env;
}

import { createEnv } from "@t3-oss/env-nextjs";
import { z } from "zod";

export const env = createEnv({
  server: {
    UNKEY_TOKEN: z.string().min(20),
    UNKEY_API_ID: z.string().min(20),
    OPEN_AI_KEY: z.string().min(20),
  },
  runtimeEnv: {
    UNKEY_TOKEN: process.env.UNKEY_TOKEN,
    UNKEY_API_ID: process.env.UNKEY_API_ID,
    OPEN_AI_KEY: process.env.OPEN_AI_KEY,
  },
});

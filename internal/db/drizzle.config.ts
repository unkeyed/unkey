import type { Config } from "drizzle-kit";

export default {
  verbose: true,
  schema: "./src/schema/index.ts",
  driver: "mysql2",
  dbCredentials: {
    uri: process.env.DRIZZLE_DATABASE_URL!,
  },
} satisfies Config;

import type { Config } from "drizzle-kit";

export default {
  verbose: true,
  strict: true,
  schema: "./src/schema/index.ts",
  out: "./drizzle",
  driver: "mysql2",
  strict: false,
  dbCredentials: {
    connectionString: process.env.DRIZZLE_DATABASE_URL!,
  },
} satisfies Config;

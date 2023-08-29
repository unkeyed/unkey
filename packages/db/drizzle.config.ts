import type { Config } from "drizzle-kit";

export default {
  schema: "./src/schema/index.ts",
  out: "./drizzle",
  driver: "mysql2",
  dbCredentials: {
    // rome-ignore lint: suspicious/noNonNullAssertion
    connectionString: process.env.DRIZZLE_DATABASE_URL!,
  },
} satisfies Config;

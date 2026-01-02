import { defineConfig } from "drizzle-kit";

export default defineConfig({
  verbose: true,
  schema: "./src/schema/index.ts",
  dialect: "mysql",
  //tablesFilter: ["acme_challenges"],
  dbCredentials: {
    // biome-ignore lint/style/noNonNullAssertion: Safe to leave
    url: process.env.DRIZZLE_DATABASE_URL!,
  },
});

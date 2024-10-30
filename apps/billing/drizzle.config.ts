import { defineConfig } from "drizzle-kit";

export default defineConfig({
  verbose: true,
  schema: "./src/lib/db-marketing/schemas/*.ts",
  out: "./drizzle",
  dialect: "mysql",
  dbCredentials: {
    host: process.env.MARKETING_DATABASE_HOST!,
    user: process.env.MARKETING_DATABASE_USERNAME!,
    password: process.env.MARKETING_DATABASE_PASSWORD!,
    database: process.env.MARKETING_DATABASE_NAME!,
    ssl: {
      rejectUnauthorized: true,
    },
  },
});

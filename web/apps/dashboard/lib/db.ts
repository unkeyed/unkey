import { dbEnv } from "@/lib/env";
import { createCommentedPool, drizzle, schema, staticTagsFromEnv, withReplicas } from "@unkey/db";

const { DATABASE_PRIMARY, DATABASE_REPLICA } = dbEnv();
const tags = staticTagsFromEnv("dashboard");

const primary = drizzle(createCommentedPool({ uri: DATABASE_PRIMARY }, tags), {
  schema,
  mode: "default",
});
const replica = DATABASE_REPLICA
  ? drizzle(createCommentedPool({ uri: DATABASE_REPLICA }, tags), { schema, mode: "default" })
  : undefined;

export const db = replica ? withReplicas(primary, [replica], () => replica) : primary;

export * from "@unkey/db";

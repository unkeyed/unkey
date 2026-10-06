import { dbEnv } from "@/lib/env";
import { createCommentedPool, drizzle, schema, staticTagsFromEnv, withReplicas } from "@unkey/db";

const { DATABASE_PRIMARY, DATABASE_REPLICA } = dbEnv();

const primary = drizzle(
  createCommentedPool({ uri: DATABASE_PRIMARY }, staticTagsFromEnv("dashboard-rw")),
  {
    schema,
    mode: "default",
  },
);
const replica = DATABASE_REPLICA
  ? drizzle(createCommentedPool({ uri: DATABASE_REPLICA }, staticTagsFromEnv("dashboard-ro")), {
      schema,
      mode: "default",
    })
  : undefined;

export const primaryDb = primary;
export const db = replica ? withReplicas(primary, [replica], () => replica) : primary;

export * from "@unkey/db";

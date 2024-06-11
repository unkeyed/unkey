import { newId } from "@unkey/id";
import { ConsoleLogger } from "@unkey/worker-logging";
import { createConnection, schema } from "../db";
import type { Env } from "../env";
import type { MessageBody } from "./message";

export async function storeMigrationError(message: MessageBody, env: Env): Promise<void> {
  const db = createConnection({
    host: env.DATABASE_HOST,
    username: env.DATABASE_USERNAME,
    password: env.DATABASE_PASSWORD,
    retry: 3,
    logger: new ConsoleLogger({ requestId: "", application: "api", environment: env.ENVIRONMENT }),
  });

  await db.insert(schema.keyMigrationErrors).values({
    id: newId("test"),
    migrationId: message.migrationId,
    workspaceId: message.workspaceId,
    message: message,
    createdAt: Date.now(),
  });
}

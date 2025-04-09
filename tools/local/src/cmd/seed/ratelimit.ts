import * as clack from "@clack/prompts";
import { eq, schema } from "@unkey/db";
import { promptForBatchSize, selectOrCreateResource, withDatabase } from "./batch-helper";
import { insertRatelimitEvents } from "./batch-operations";
import { clickhouse, connectDatabase, generateRandomString } from "./utils";

const DEFAULT_BATCH_SIZE = 50_000;

/**
 * Get existing ratelimit namespaces for the workspace
 */
async function getRatelimitNamespaces(workspaceId: string) {
  return withDatabase(async (db) => {
    return db
      .select({
        id: schema.ratelimitNamespaces.id,
        name: schema.ratelimitNamespaces.name,
      })
      .from(schema.ratelimitNamespaces)
      .where(eq(schema.ratelimitNamespaces.workspaceId, workspaceId))
      .orderBy(schema.ratelimitNamespaces.name);
  }, connectDatabase);
}

/**
 * Create a new ratelimit namespace
 */
async function createRatelimitNamespace(workspaceId: string, name: string) {
  return withDatabase(async (db) => {
    const namespaceId = `rlns_${generateRandomString(24)}`;
    const namespace = {
      id: namespaceId,
      workspaceId: workspaceId,
      name: name,
      createdAtM: Date.now(),
      updatedAtM: Date.now(),
    };
    await db.insert(schema.ratelimitNamespaces).values(namespace);

    return namespaceId;
  }, connectDatabase);
}

/**
 * Main function to seed ratelimit data
 */
export async function seedRatelimitData(workspaceId: string, count: number) {
  // Get existing namespaces
  const existingNamespaces = await getRatelimitNamespaces(workspaceId);

  // Use the utility to select or create a resource
  const { id: namespaceId, name: namespaceName } = await selectOrCreateResource(
    "namespace",
    existingNamespaces,
    (name) => createRatelimitNamespace(workspaceId, name),
    `Ratelimit Namespace ${new Date().toISOString().substring(0, 10)}`,
  );

  // Prompt for API request generation
  const generateApiLogs = await clack.confirm({
    message: "Would you like to generate matching API request logs for the ratelimit events?",
    initialValue: true,
  });

  if (clack.isCancel(generateApiLogs)) {
    clack.cancel("Operation cancelled");
    process.exit(0);
  }

  // Configure batch size using utility
  const batchSize = await promptForBatchSize(DEFAULT_BATCH_SIZE);

  // Insert ratelimit events with the batched operation utility
  const result = await insertRatelimitEvents(
    clickhouse,
    workspaceId,
    namespaceId,
    count,
    generateApiLogs as boolean,
    batchSize,
  );

  return {
    namespaceId,
    namespaceName,
    eventCount: count,
    batchSize,
    performance: result,
  };
}

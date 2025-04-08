import * as clack from "@clack/prompts";
import { eq, schema } from "@unkey/db";
import {
  clickhouse,
  connectDatabase,
  generateRandomApiRequest,
  generateRandomString,
  generateTimestamp,
  generateUuid,
} from "./utils";

const BATCH_SIZE = 50_000;

async function getRatelimitNamespaces(workspaceId: string) {
  const { db, conn } = await connectDatabase();

  try {
    const namespaces = await db
      .select({
        id: schema.ratelimitNamespaces.id,
        name: schema.ratelimitNamespaces.name,
      })
      .from(schema.ratelimitNamespaces)
      .where(eq(schema.ratelimitNamespaces.workspaceId, workspaceId))
      .orderBy(schema.ratelimitNamespaces.name);

    return namespaces;
  } finally {
    await conn.end();
  }
}

async function createRatelimitNamespace(workspaceId: string, name: string) {
  const { db, conn } = await connectDatabase();

  try {
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
  } finally {
    await conn.end();
  }
}

// Generate a random ratelimit event
function generateRatelimitEvent(workspaceId: string, namespaceId: string, requestId?: string) {
  // Generate a random timestamp with the same distribution as API requests
  const time = generateTimestamp();

  const commonPrefixes = ["ip:", "user:", "tenant:", "org:", "key:"];
  const prefix = commonPrefixes[Math.floor(Math.random() * commonPrefixes.length)];
  const identifier = `${prefix}${generateRandomString(12)}`;

  // Determine if this request passed the rate limit
  const passed = Math.random() < 0.9; // 90% of requests pass rate limiting

  // Random values for rate limit tracking
  const remaining = passed ? Math.floor(Math.random() * 1000) : 0;
  const limit = 500 + Math.floor(Math.random() * 9500); // Between 500 and 10000
  const reset = Date.now() + 60 * 1000; // Reset in 1 minute

  // Generate a request ID if not provided
  const generated_request_id = requestId || generateUuid();

  return {
    request_id: generated_request_id,
    time,
    workspace_id: workspaceId,
    namespace_id: namespaceId,
    identifier,
    passed,
    remaining,
    limit,
    reset,
  };
}

async function insertRatelimitEvents(
  workspaceId: string,
  namespaceId: string,
  count: number,
  generateMatchingApiRequests = true,
) {
  const spinner = clack.spinner();
  spinner.start(
    `Preparing to insert ${count} ratelimit events for workspace ${workspaceId} and namespace ${namespaceId}`,
  );

  const doRatelimitInsert = clickhouse.ratelimits.insert;
  const doApiInsert = clickhouse.api.insert;
  let insertedCount = 0;
  let batchNumber = 0;

  try {
    while (insertedCount < count) {
      batchNumber++;
      const batchSize = Math.min(BATCH_SIZE, count - insertedCount);
      spinner.message(
        `Generating batch ${batchNumber} with realistic time distribution (${batchSize} records)...`,
      );

      const batchOfRatelimitRecords = [];
      const batchOfApiRequestRecords = [];

      for (let i = 0; i < batchSize; i++) {
        // For some ratelimit events, we want to create matching API request logs
        const createApiRequestLog = generateMatchingApiRequests && Math.random() < 0.8; // 80% chance
        const requestId = generateUuid();

        // Create the ratelimit event
        const ratelimitEvent = generateRatelimitEvent(workspaceId, namespaceId, requestId);

        batchOfRatelimitRecords.push(ratelimitEvent);

        // If needed, create a matching API request
        if (createApiRequestLog) {
          const apiRequest = generateRandomApiRequest(workspaceId);
          // Override the generated request_id with our requestId to link the records
          apiRequest.request_id = requestId;
          // Match the timestamp
          apiRequest.time = ratelimitEvent.time;
          // Set response status based on if it passed rate limiting
          if (!ratelimitEvent.passed) {
            apiRequest.response_status = 429; // Too Many Requests
            apiRequest.error = "Rate limit exceeded";
            apiRequest.response_body = JSON.stringify({
              error: {
                code: "rate_limit_exceeded",
                message: "You have exceeded the rate limit for this endpoint",
                details: {
                  limit: ratelimitEvent.limit,
                  reset: ratelimitEvent.reset,
                },
              },
            });
          }

          batchOfApiRequestRecords.push(apiRequest);
        }
      }

      if (batchOfRatelimitRecords.length > 0) {
        spinner.message(
          `Inserting ${batchOfRatelimitRecords.length} ratelimit events (batch ${batchNumber})...`,
        );

        await doRatelimitInsert(batchOfRatelimitRecords);

        if (batchOfApiRequestRecords.length > 0) {
          spinner.message(
            `Inserting ${batchOfApiRequestRecords.length} matching API request logs...`,
          );

          await doApiInsert(batchOfApiRequestRecords);
        }
      }

      insertedCount += batchSize;
      if (batchNumber % 5 === 0 || insertedCount === count) {
        spinner.message(`Processed ${insertedCount}/${count} records...`);
      }
    }

    spinner.stop(
      `Successfully inserted ${count} ratelimit events with ${
        generateMatchingApiRequests ? "matching API requests" : "no matching API requests"
      }.`,
    );
  } catch (error: any) {
    spinner.stop(`Error inserting data during batch ${batchNumber}: ${error.message}`);
    console.error("ClickHouse Insert Error Details:", error);
    throw error;
  }
}

export async function seedRatelimitData(workspaceId: string, count: number) {
  // First, fetch existing ratelimit namespaces
  const spinner = clack.spinner();
  spinner.start(`Fetching existing ratelimit namespaces for workspace ${workspaceId}...`);

  const existingNamespaces = await getRatelimitNamespaces(workspaceId);
  spinner.stop(`Found ${existingNamespaces.length} existing ratelimit namespaces.`);

  // Ask user if they want to use an existing namespace or create a new one
  let namespaceId: string;
  let namespaceName: string;

  if (existingNamespaces.length > 0) {
    const namespaceChoice = await clack.select({
      message: "Would you like to use an existing namespace or create a new one?",
      options: [
        { value: "existing", label: "Use an existing namespace" },
        { value: "new", label: "Create a new namespace" },
      ],
    });

    if (clack.isCancel(namespaceChoice)) {
      clack.cancel("Operation cancelled");
      process.exit(0);
    }

    if (namespaceChoice === "existing") {
      // Get user to select an existing namespace
      const selectedNamespace = await clack.select({
        message: "Select a namespace:",
        options: existingNamespaces.map((ns) => ({
          value: ns.id,
          label: ns.name,
        })),
      });

      if (clack.isCancel(selectedNamespace)) {
        clack.cancel("Operation cancelled");
        process.exit(0);
      }

      namespaceId = selectedNamespace as string;
      namespaceName = existingNamespaces.find((ns) => ns.id === namespaceId)?.name || "Unknown";

      spinner.start(`Using existing namespace: ${namespaceName} (${namespaceId})`);
      spinner.stop();
    } else {
      // Create a new namespace
      namespaceName = (await clack.text({
        message: "Enter a name for the new ratelimit namespace:",
        defaultValue: `Ratelimit Namespace ${new Date().toISOString().substring(0, 10)}`,
        validate(value) {
          if (!value || value.trim().length === 0) {
            return "Please enter a valid namespace name";
          }
        },
      })) as string;

      if (clack.isCancel(namespaceName)) {
        clack.cancel("Operation cancelled");
        process.exit(0);
      }

      spinner.start(`Creating ratelimit namespace "${namespaceName}"...`);
      namespaceId = await createRatelimitNamespace(workspaceId, namespaceName as string);
      spinner.stop(`Created ratelimit namespace: ${namespaceName} (${namespaceId})`);
    }
  } else {
    // No existing namespaces, create a new one
    namespaceName = (await clack.text({
      message: "No existing namespaces found. Enter a name for the new ratelimit namespace:",
      defaultValue: `Ratelimit Namespace ${new Date().toISOString().substring(0, 10)}`,
      validate(value) {
        if (!value || value.trim().length === 0) {
          return "Please enter a valid namespace name";
        }
      },
    })) as string;

    if (clack.isCancel(namespaceName)) {
      clack.cancel("Operation cancelled");
      process.exit(0);
    }

    spinner.start(`Creating ratelimit namespace "${namespaceName}"...`);
    namespaceId = await createRatelimitNamespace(workspaceId, namespaceName as string);
    spinner.stop(`Created ratelimit namespace: ${namespaceName} (${namespaceId})`);
  }

  const generateApiLogs = await clack.confirm({
    message: "Would you like to generate matching API request logs for the ratelimit events?",
    initialValue: true,
  });

  if (clack.isCancel(generateApiLogs)) {
    clack.cancel("Operation cancelled");
    process.exit(0);
  }

  await insertRatelimitEvents(workspaceId, namespaceId, count, generateApiLogs as boolean);

  return {
    namespaceId,
    namespaceName,
    eventCount: count,
  };
}

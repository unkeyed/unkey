import crypto from "node:crypto";
import * as clack from "@clack/prompts";
import { and, eq, isNull, schema } from "@unkey/db";
import {
  clickhouse,
  connectDatabase,
  generateMetadata,
  generateRandomApiRequest,
  generateRandomString,
  generateTimestamp,
  generateUuid,
} from "./utils";

const BATCH_SIZE = 50_000;

async function getAPIs(workspaceId: string) {
  const { db, conn } = await connectDatabase();

  try {
    const apis = await db
      .select({
        id: schema.apis.id,
        name: schema.apis.name,
        keyAuthId: schema.apis.keyAuthId,
      })
      .from(schema.apis)
      .where(and(eq(schema.apis.workspaceId, workspaceId), isNull(schema.apis.deletedAtM)))
      .orderBy(schema.apis.name);

    return apis;
  } finally {
    await conn.end();
  }
}

async function createApi(workspaceId: string, name: string) {
  const { db, conn } = await connectDatabase();

  try {
    const keyAuthId = `kauth_${generateRandomString(24)}`;
    const payload = {
      id: keyAuthId,
      workspaceId: workspaceId,
      createdAt: Date.now(),
      updatedAt: Date.now(),
      storeEncryptedKeys: Math.random() > 0.7, // 30% chance to store encrypted keys
      defaultPrefix: `uk_${generateRandomString(2)}`,
      defaultBytes: 16,
      sizeApprox: 0,
      sizeLastUpdatedAt: Date.now(),
    };
    await db.insert(schema.keyAuth).values(payload);

    // Then create the API record
    const apiId = `api_${generateRandomString(24)}`;
    const apisPayload = {
      id: apiId,
      name: name,
      workspaceId: workspaceId,
      ipWhitelist: Math.random() > 0.8 ? "192.168.1.1,10.0.0.1" : null, // 20% chance to have IP whitelist
      authType: "key",
      keyAuthId: keyAuthId,
      createdAt: Date.now(),
      updatedAt: Date.now(),
      deletedAt: null,
      deletedBy: null,
    };
    //@ts-expect-error I don't know why authType type doesn't match with match schema but it works
    await db.insert(schema.apis).values(apisPayload);

    return { apiId, keyAuthId };
  } finally {
    await conn.end();
  }
}

// Generate a hash for a key
function generateKeyHash(keyContent: string) {
  return crypto.createHash("sha256").update(keyContent).digest("hex");
}

// Generate a random key name
function generateKeyName() {
  const environments = ["Development", "Staging", "Production", "Testing"];
  const purposes = [
    "API Access",
    "Admin",
    "Readonly",
    "Server",
    "Client",
    "Backend",
    "Frontend",
    "Mobile",
  ];
  const environment = environments[Math.floor(Math.random() * environments.length)];
  const purpose = purposes[Math.floor(Math.random() * purposes.length)];

  if (Math.random() > 0.5) {
    return `${environment} ${purpose} Key`;
  }
  return `${purpose} Key for ${environment}`;
}

// Create keys for an API
async function createKeysForApi(
  workspaceId: string,
  apiId: string,
  keyAuthId: string,
  count: number,
) {
  const { db, conn } = await connectDatabase();
  const createdKeys = [];

  try {
    const spinner = clack.spinner();
    spinner.start(`Creating ${count} keys for API ${apiId}`);

    for (let i = 0; i < count; i++) {
      // Generate key details
      const keyId = `key_${generateRandomString(24)}`;
      const keyPrefix = `uk_${generateRandomString(3)}`;
      const fullKeyContent = `${keyPrefix}_${generateRandomString(32)}`;
      const keyHash = generateKeyHash(fullKeyContent);

      // Determine key attributes with realistic distributions
      const name = generateKeyName();
      const ownerId = `user_${generateRandomString(16)}`;
      const meta = Math.random() > 0.3 ? JSON.stringify(generateMetadata()) : null; // 70% chance to have metadata

      // Expiration: 25% of keys expire within a year
      const expires =
        Math.random() > 0.75
          ? new Date(Date.now() + Math.floor(Math.random() * 365) * 86400000)
          : null;

      // Rate limiting: 40% of keys have rate limits
      const hasRatelimit = Math.random() > 0.6;
      const ratelimitLimit = hasRatelimit ? 100 * (1 + Math.floor(Math.random() * 100)) : null;
      const ratelimitDuration = hasRatelimit
        ? 60 * 1000 * (1 + Math.floor(Math.random() * 60))
        : null;
      const ratelimitAsync = hasRatelimit ? Math.random() > 0.7 : null;

      // Usage limits: 20% of keys have usage limits
      const hasUsageLimit = Math.random() > 0.8;
      const remaining = hasUsageLimit ? 1000 * (1 + Math.floor(Math.random() * 100)) : null;

      // Environment
      const environment =
        Math.random() > 0.6
          ? ["production", "development", "test", "staging"][Math.floor(Math.random() * 4)]
          : null;

      // Enabled/disabled: 95% of keys are enabled
      const enabled = Math.random() > 0.05;

      const payload = {
        id: keyId,
        keyAuthId: keyAuthId,
        hash: keyHash,
        start: keyPrefix,
        workspaceId: workspaceId,
        forWorkspaceId: null,
        name: name,
        ownerId: ownerId,
        identityId: null,
        meta: meta,
        expires: expires,
        createdAt: Date.now(),
        updatedAt: Date.now(),
        deletedAt: null,
        deletedBy: null,
        refillDay: null,
        refillAmount: null,
        lastRefillAt: null,
        enabled: enabled,
        remaining: remaining,
        ratelimitAsync: ratelimitAsync,
        ratelimitLimit: ratelimitLimit,
        ratelimitDuration: ratelimitDuration,
        environment: environment,
      };

      // Insert the key
      await db.insert(schema.keys).values(payload);

      createdKeys.push({
        id: keyId,
        name: name,
        prefix: keyPrefix,
        enabled: enabled,
        hasRatelimit,
        hasUsageLimit,
      });

      // Update the size approximation in keyAuth
      if (i === count - 1) {
        await db
          .update(schema.keyAuth)
          .set({
            sizeApprox: count,
            sizeLastUpdatedAt: Date.now(),
          })
          .where(eq(schema.keyAuth.id, keyAuthId));
      }
    }

    spinner.stop(`Created ${count} keys for API ${apiId}`);
    return createdKeys;
  } finally {
    await conn.end();
  }
}

// Get existing keys for an API/keyspace
async function getExistingKeys(workspaceId: string, keyAuthId: string) {
  const { db, conn } = await connectDatabase();

  try {
    const keys = await db
      .select({
        id: schema.keys.id,
        name: schema.keys.name,
        start: schema.keys.start,
        enabled: schema.keys.enabled,
        ratelimitLimit: schema.keys.ratelimitLimit,
        remaining: schema.keys.remaining,
      })
      .from(schema.keys)
      .where(
        and(
          eq(schema.keys.workspaceId, workspaceId),
          eq(schema.keys.keyAuthId, keyAuthId),
          isNull(schema.keys.deletedAtM),
        ),
      );

    return keys.map((key) => ({
      id: key.id,
      name: key.name ?? `Unnamed Key (${key.id})`,
      prefix: key.start,
      enabled: Boolean(key.enabled),
      hasRatelimit: key.ratelimitLimit !== null,
      hasUsageLimit: key.remaining !== null,
    }));
  } finally {
    await conn.end();
  }
}

// Generate a random verification event
function generateVerificationEvent(
  workspaceId: string,
  keyspaceId: string,
  keyId: string,
  requestId?: string,
) {
  // Generate a timestamp using the same distribution logic as API requests
  const time = generateTimestamp();

  // Outcomes with realistic distribution
  const outcomeDistribution = Math.random();
  let outcome: string;
  if (outcomeDistribution < 0.85) {
    outcome = "VALID"; // 85% valid
  } else if (outcomeDistribution < 0.9) {
    outcome = "RATE_LIMITED"; // 5% rate limited
  } else if (outcomeDistribution < 0.93) {
    outcome = "EXPIRED"; // 3% expired
  } else if (outcomeDistribution < 0.95) {
    outcome = "DISABLED"; // 2% disabled
  } else if (outcomeDistribution < 0.97) {
    outcome = "FORBIDDEN"; // 2% forbidden
  } else if (outcomeDistribution < 0.99) {
    outcome = "USAGE_EXCEEDED"; // 2% usage exceeded
  } else {
    outcome = "INSUFFICIENT_PERMISSIONS"; // 1% insufficient permissions
  }

  // Generate tags (common categories for segmenting key verifications)
  const tagOptions = [
    "api",
    "web",
    "mobile",
    "server",
    "client",
    "frontend",
    "backend",
    "test",
    "prod",
  ];
  const tagCount = Math.floor(Math.random() * 3); // 0-2 tags
  const tags: string[] = [];

  for (let i = 0; i < tagCount; i++) {
    const tag = tagOptions[Math.floor(Math.random() * tagOptions.length)];
    if (!tags.includes(tag)) {
      tags.push(tag);
    }
  }

  // Sort tags to match expected format
  tags.sort();

  // Generate request ID if not provided
  const generatedRequestId = requestId || generateUuid();

  // Generate a region
  const regions = ["us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1", "sa-east-1"];
  const region = regions[Math.floor(Math.random() * regions.length)];

  // Optionally include identity_id (30% of the time)
  const identityId = Math.random() < 0.3 ? `ident_${generateRandomString(24)}` : "";

  return {
    request_id: generatedRequestId,
    time,
    workspace_id: workspaceId,
    key_space_id: keyspaceId,
    key_id: keyId,
    region,
    tags,
    outcome,
    identity_id: identityId,
  };
}

// Insert verification events into ClickHouse
async function insertVerificationEvents(
  workspaceId: string,
  keyAuthId: string,
  keys: {
    id: string;
    name: string;
    prefix: string;
    enabled: boolean;
    hasRatelimit: boolean;
    hasUsageLimit: boolean;
  }[],
  count: number,
  generateMatchingApiRequests = true,
) {
  const spinner = clack.spinner();
  spinner.start(`Preparing to insert ${count} verification events for workspace ${workspaceId}`);

  const doVerificationInsert = clickhouse.verifications.insert;
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

      const batchOfVerificationRecords = [];
      const batchOfApiRequestRecords = [];

      for (let i = 0; i < batchSize; i++) {
        // Select a random key from our created keys
        const randomKeyIndex = Math.floor(Math.random() * keys.length);
        const key = keys[randomKeyIndex];

        // For some verification events, we want to create matching API request logs
        const createApiRequestLog = generateMatchingApiRequests && Math.random() < 0.8; // 80% chance
        const requestId = generateUuid();

        // Create the verification event, biasing outcomes based on key properties
        let verificationEvent: any;

        // If key is disabled, higher chance of DISABLED outcome
        if (!key.enabled && Math.random() < 0.6) {
          verificationEvent = generateVerificationEvent(workspaceId, keyAuthId, key.id, requestId);
          verificationEvent.outcome = "DISABLED";
        }
        // If key has rate limit, higher chance of RATE_LIMITED outcome
        else if (key.hasRatelimit && Math.random() < 0.3) {
          verificationEvent = generateVerificationEvent(workspaceId, keyAuthId, key.id, requestId);
          verificationEvent.outcome = "RATE_LIMITED";
        }
        // If key has usage limit, higher chance of USAGE_EXCEEDED outcome
        else if (key.hasUsageLimit && Math.random() < 0.2) {
          verificationEvent = generateVerificationEvent(workspaceId, keyAuthId, key.id, requestId);
          verificationEvent.outcome = "USAGE_EXCEEDED";
        }
        // Otherwise generate a regular event
        else {
          verificationEvent = generateVerificationEvent(workspaceId, keyAuthId, key.id, requestId);
        }

        batchOfVerificationRecords.push(verificationEvent);

        // If needed, create a matching API request
        if (createApiRequestLog) {
          const apiRequest = generateRandomApiRequest(workspaceId);
          // Override the generated request_id with our requestId to link the records
          apiRequest.request_id = requestId;
          // Match the timestamp
          apiRequest.time = verificationEvent.time;
          // Ensure we're using the key verification endpoint
          apiRequest.path = "/v1/keys.verify";

          // Set key information in request body
          const requestBody = {
            key: `${key.prefix}_${generateRandomString(32)}`,
            apiId: `api_${generateRandomString(24)}`,
          };
          apiRequest.request_body = JSON.stringify(requestBody);

          // Set response status based on verification outcome
          if (verificationEvent.outcome !== "VALID") {
            switch (verificationEvent.outcome) {
              case "RATE_LIMITED":
                apiRequest.response_status = 429; // Too Many Requests
                apiRequest.error = "Rate limit exceeded";
                break;
              case "EXPIRED":
              case "DISABLED":
              case "FORBIDDEN":
              case "USAGE_EXCEEDED":
              case "INSUFFICIENT_PERMISSIONS":
                apiRequest.response_status = 401; // Unauthorized
                apiRequest.error = verificationEvent.outcome.toLowerCase().replace("_", " ");
                break;
            }

            apiRequest.response_body = JSON.stringify({
              valid: false,
              code: String(apiRequest.response_status),
              message: apiRequest.error,
            });
          } else {
            // Valid response
            apiRequest.response_status = 200;
            apiRequest.response_body = JSON.stringify({
              valid: true,
              keyId: key.id,
              ownerId: `user_${generateRandomString(16)}`,
              meta: Math.random() < 0.7 ? generateMetadata() : undefined,
            });
          }

          batchOfApiRequestRecords.push(apiRequest);
        }
      }

      if (batchOfVerificationRecords.length > 0) {
        spinner.message(
          `Inserting ${batchOfVerificationRecords.length} verification events (batch ${batchNumber})...`,
        );

        // Insert verification events
        await doVerificationInsert(batchOfVerificationRecords);

        // Insert matching API requests if any
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
      `Successfully inserted ${count} verification events with ${
        generateMatchingApiRequests ? "matching API requests" : "no matching API requests"
      }.`,
    );
  } catch (error: any) {
    spinner.stop(`Error inserting data during batch ${batchNumber}: ${error.message}`);
    console.error("ClickHouse Insert Error Details:", error);
    throw error;
  }
}

export async function seedApiAndKeys(workspaceId: string, count: number) {
  // First, fetch existing APIs
  const spinner = clack.spinner();
  spinner.start(`Fetching existing APIs for workspace ${workspaceId}...`);

  const existingApis = await getAPIs(workspaceId);
  spinner.stop(`Found ${existingApis.length} existing APIs.`);

  // Variables to track API and keyspace IDs
  let apiId: string;
  let keyAuthId: string;
  let apiName: string;

  // Ask user if they want to use an existing API or create a new one
  if (existingApis.length > 0) {
    const apiChoice = await clack.select({
      message: "Would you like to use an existing API or create a new one?",
      options: [
        { value: "existing", label: "Use an existing API" },
        { value: "new", label: "Create a new API" },
      ],
    });

    if (clack.isCancel(apiChoice)) {
      clack.cancel("Operation cancelled");
      process.exit(0);
    }

    if (apiChoice === "existing") {
      // Get user to select an existing API
      const selectedApi = await clack.select({
        message: "Select an API:",
        options: existingApis.map((api) => ({
          value: JSON.stringify({ id: api.id, keyAuthId: api.keyAuthId }),
          label: api.name,
        })),
      });

      if (clack.isCancel(selectedApi)) {
        clack.cancel("Operation cancelled");
        process.exit(0);
      }

      const { id, keyAuthId: selectedKeyAuthId } = JSON.parse(selectedApi as string);
      apiId = id;
      keyAuthId = selectedKeyAuthId;
      apiName = existingApis.find((api) => api.id === apiId)?.name || "Unknown";

      spinner.start(`Using existing API: ${apiName} (${apiId})`);
      spinner.stop();
    } else {
      // Create a new API
      apiName = (await clack.text({
        message: "Enter a name for the new API:",
        defaultValue: `API ${new Date().toISOString().substring(0, 10)}`,
        validate(value) {
          if (!value || value.trim().length === 0) {
            return "Please enter a valid API name";
          }
        },
      })) as string;

      if (clack.isCancel(apiName)) {
        clack.cancel("Operation cancelled");
        process.exit(0);
      }

      spinner.start(`Creating API "${apiName}"...`);
      const result = await createApi(workspaceId, apiName as string);
      apiId = result.apiId;
      keyAuthId = result.keyAuthId;
      spinner.stop(`Created API: ${apiName} (${apiId}) with keyspace: ${keyAuthId}`);
    }
  } else {
    // No existing APIs, create a new one
    apiName = (await clack.text({
      message: "No existing APIs found. Enter a name for the new API:",
      defaultValue: `API ${new Date().toISOString().substring(0, 10)}`,
      validate(value) {
        if (!value || value.trim().length === 0) {
          return "Please enter a valid API name";
        }
      },
    })) as string;

    if (clack.isCancel(apiName)) {
      clack.cancel("Operation cancelled");
      process.exit(0);
    }

    spinner.start(`Creating API "${apiName}"...`);
    const result = await createApi(workspaceId, apiName as string);
    apiId = result.apiId;
    keyAuthId = result.keyAuthId;
    spinner.stop(`Created API: ${apiName} (${apiId}) with keyspace: ${keyAuthId}`);
  }

  // Check if the API already has keys
  spinner.start(`Checking for existing keys for API ${apiName}...`);
  const existingKeys = await getExistingKeys(workspaceId, keyAuthId);
  spinner.stop(`Found ${existingKeys.length} existing keys for API ${apiName}.`);

  let keys: {
    id: string;
    name: string;
    prefix: string;
    enabled: boolean;
    hasRatelimit: boolean;
    hasUsageLimit: boolean;
  }[] = existingKeys;

  // Ask if the user wants to create additional keys if using an existing API with keys
  if (existingKeys.length > 0) {
    const createMoreKeys = await clack.confirm({
      message: `API ${apiName} already has ${existingKeys.length} keys. Would you like to create additional keys?`,
      initialValue: false,
    });

    if (clack.isCancel(createMoreKeys)) {
      clack.cancel("Operation cancelled");
      process.exit(0);
    }

    if (createMoreKeys) {
      // Ask how many additional keys to create
      const keyCount = await clack.text({
        message: "How many additional keys would you like to create for this API?",
        defaultValue: "100",
        validate(value) {
          const num = Number.parseInt(value, 10);
          if (Number.isNaN(num) || num <= 0) {
            return "Please enter a valid positive number";
          }
        },
      });

      if (clack.isCancel(keyCount)) {
        clack.cancel("Operation cancelled");
        process.exit(0);
      }

      const numKeyCount = Number.parseInt(keyCount as string, 10);
      const newKeys = await createKeysForApi(workspaceId, apiId, keyAuthId, numKeyCount);

      // Add the new keys to our existing keys array
      keys = [...existingKeys, ...newKeys];
    }
  } else {
    // Either a new API or an existing API with no keys
    // Ask how many keys to create
    const keyCount = await clack.text({
      message: "How many keys would you like to create for this API?",
      defaultValue: "100",
      validate(value) {
        const num = Number.parseInt(value, 10);
        if (Number.isNaN(num) || num <= 0) {
          return "Please enter a valid positive number";
        }
      },
    });

    if (clack.isCancel(keyCount)) {
      clack.cancel("Operation cancelled");
      process.exit(0);
    }

    const numKeyCount = Number.parseInt(keyCount as string, 10);
    keys = await createKeysForApi(workspaceId, apiId, keyAuthId, numKeyCount);
  }

  // Check if we have keys before offering to generate verification events
  if (keys.length > 0) {
    const generateVerifications = await clack.confirm({
      message: "Would you like to generate key verification events?",
      initialValue: true,
    });

    if (clack.isCancel(generateVerifications)) {
      clack.cancel("Operation cancelled");
      process.exit(0);
    }

    if (generateVerifications) {
      const verificationCount = await clack.text({
        message: "How many verification events would you like to create?",
        defaultValue: count.toString(),
        validate(value) {
          const num = Number.parseInt(value, 10);
          if (Number.isNaN(num) || num <= 0) {
            return "Please enter a valid positive number";
          }
        },
      });

      if (clack.isCancel(verificationCount)) {
        clack.cancel("Operation cancelled");
        process.exit(0);
      }

      const numVerificationCount = Number.parseInt(verificationCount as string, 10);

      const generateApiLogs = await clack.confirm({
        message:
          "Would you like to generate matching API request logs for the verification events?",
        initialValue: true,
      });

      if (clack.isCancel(generateApiLogs)) {
        clack.cancel("Operation cancelled");
        process.exit(0);
      }

      await insertVerificationEvents(
        workspaceId,
        keyAuthId,
        keys,
        numVerificationCount,
        generateApiLogs as boolean,
      );
    }
  } else {
    // biome-ignore lint/suspicious/noConsoleLog: <explanation>
    console.log(
      "No keys available for verification events. Skipping verification event generation.",
    );
  }

  return {
    apiId,
    apiName,
    keyAuthId,
    keyCount: keys.length,
  };
}
